package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/jonfriesen/subscriber-tracker-api/model"
	"github.com/jonfriesen/subscriber-tracker-api/storage/postgresql"
	"github.com/rs/cors"
)

var (
	ErrNotFound               = errors.New("Error: not found")
	ErrRquestTypeNotSupported = errors.New("Error: HTTP Request type not supported")
)

// Database is a dirty global variable for the DB conn pool.
var Database *postgresql.PostgreSQL

// MissingDB is a premade struct with a known format.
type MissingDB struct {
	ErrorMessage string `json:"error_message,omitempty"`
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

// Handler handles http requests
type Handler struct {
	handler http.Handler
	wsHub   *Hub
}

// New creates a new http handler
func New() Handler {
	mux := http.NewServeMux()

	h := Handler{}

	h.wsHub = &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
	go h.wsHub.Run()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Oh hey there 👋! You're probably looking for /subscribers/")
	})
	mux.Handle("/subscribers/", wrapper(h.list))
	mux.HandleFunc("/substream", func(w http.ResponseWriter, r *http.Request) {
		serveWs(h.wsHub, w, r)
	})

	corsHandler := cors.Default().Handler(mux)
	h.handler = corsHandler

	return h
}

// Get returns the http.Handler
func (h *Handler) Get() http.Handler {
	return h.handler
}

func (h *Handler) list(w io.Writer, r *http.Request) (interface{}, int, error) {
	reqDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Println("Error dumping http request", err.Error())
	}
	log.Println(string(reqDump))

	switch r.Method {
	case "GET":

		if Database == nil {
			return MissingDB{ErrorMessage: "Database appears to be missing. These changes will not be saved."}, http.StatusOK, nil
		}

		v, err := Database.ListSubscribers()
		if err != nil {
			log.Println("Database Error (LIST):", err.Error())
			return nil, http.StatusNotFound, ErrNotFound
		}

		return v, http.StatusOK, nil
	case "POST":

		if Database == nil {
			return MissingDB{ErrorMessage: "Database appears to be missing. These changes will not be saved."}, http.StatusOK, nil
		}

		var sub *model.Subscriber
		if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
			return nil, http.StatusBadRequest, err
		}

		v, err := Database.AddSubscriber(sub)
		if err != nil {
			log.Println("Database Error (ADD):", err.Error())
			return nil, http.StatusInternalServerError, err
		}

		vB, err := json.Marshal(v)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}

		h.wsHub.broadcast <- vB

		return v, http.StatusCreated, nil
	}

	return nil, http.StatusBadRequest, ErrRquestTypeNotSupported
}

func wrapper(f func(io.Writer, *http.Request) (interface{}, int, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, status, err := f(w, r)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		w.Header().Set("Content-Type", " application/json")
		w.WriteHeader(status)

		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// Run starts the Hub Router
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
