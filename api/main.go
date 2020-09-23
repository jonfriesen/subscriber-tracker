package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/jonfriesen/subscriber-tracker-api/storage/postgresql"

	"github.com/jonfriesen/subscriber-tracker-api/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:mysecretpassword@localhost:5432/postgres?sslmode=disable"
	}

	handler := api.New()
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", host, port),
		Handler: handler.Get(),
	}

	wg := new(sync.WaitGroup)
	wg.Add(2)

	go func() {
		sigquit := make(chan os.Signal, 1)
		signal.Notify(sigquit, os.Interrupt, os.Kill)

		sig := <-sigquit
		log.Printf("caught sig: %+v", sig)
		log.Printf("Gracefully shutting down server...")

		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("Unable to shut down server: %v", err)
		} else {
			log.Println("Server stopped")
		}
		wg.Done()
		wg.Done()
	}()

	go func() {
		log.Printf("Magic is happening on port %s", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("%v", err)
			wg.Done()
		} else {
			log.Println("Server closed")
			wg.Done()
		}
	}()

	wg.Add(1)
	go func() {
		log.Println("Trying to get DB Connection")
		for {
			adb, err := postgresql.NewConnection(dbURL)
			if err != nil {
				log.Println(">> No DB - Sleeping for 1 second")
				time.Sleep(1 * time.Second)
				continue
			}
			log.Println(">>> Found Database!")
			api.Database = adb
			wg.Done()
			break
		}
	}()

	wg.Wait()
}
