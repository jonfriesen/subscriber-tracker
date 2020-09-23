package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"time"

	randomdata "github.com/Pallinder/go-randomdata"
	"github.com/jonfriesen/subscriber-tracker-worker/model"
	"github.com/pkg/profile"
)

var (
	apiPath      = "http://localhost:8080"
	internalPath = "http://api"
)

func main() {
	defer profile.Start(profile.MemProfile).Stop()
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		log.Println("Attempting to connect to internal api", internalPath)
		resp, err := http.Get(internalPath)
		if err == nil && resp.StatusCode == http.StatusOK {
			log.Println("Internal Path Success", internalPath)
			respByte, err := httputil.DumpResponse(resp, true)
			if err != nil {
				log.Println("Error dumping response", err.Error())
			}
			log.Println(string(respByte))
			apiPath = internalPath
		}
	} else {
		apiPath = domain
		env := os.Getenv("ENVIRONMENT")
		if env == "" || env == "production" {
			apiPath += "/api"
		}
	}

	sigquit := make(chan os.Signal, 1)
	signal.Notify(sigquit, os.Interrupt, os.Kill)
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-sigquit:
			os.Exit(0)
		case <-ticker.C:
		}
		newSub := &model.Subscriber{
			Name:  randomdata.FullName(randomdata.RandomGender),
			Email: randomdata.Email(),
		}
		newSubB, err := json.Marshal(newSub)
		if err != nil {
			log.Println("Error marshalling generated sub.")
			return
		}
		hostPath := apiPath + "/subscribers/"

		resp, err := http.Post(hostPath, "application/json", bytes.NewBuffer(newSubB))
		if err != nil {
			log.Printf("Error occurred, ignoring: %s\n", err.Error())
			time.Sleep(1 * time.Minute)
			continue
		}
		defer resp.Body.Close()

		log.Printf("%s @ Added: %s <%s>", hostPath, newSub.Name, newSub.Email)
	}
}
