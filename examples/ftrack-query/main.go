package main

import (
	"flag"
	"github.com/conducte/ftrack-golang-api/ftrack"
	"log"
)

func main() {
	apiKey := flag.String("api_key", "", "Ftrack Api Key from Settings -> Api Keys")
	apiUser := flag.String("api_user", "", "Ftrack Api User username from enabled user")
	serverUrl := flag.String("server_url", "", "Ftrack Server Url server url eg https://ftrack.com")
	flag.Parse()
	// Construct Session from command line arguments
	session, err := ftrack.NewSession(ftrack.SessionConfig{
		ApiKey:    *apiKey,
		ApiUser:   *apiUser,
		ServerUrl: *serverUrl,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Query single Task from server
	result, err := session.Query("select name, parent.project from Task limit 1")
	if err != nil {
		log.Fatal(err)
	}
	task := result.Data[0]
	log.Println("Task: ", task)
}
