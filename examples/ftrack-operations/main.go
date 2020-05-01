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

	session, err := ftrack.NewSession(ftrack.SessionConfig{
		ApiKey:    *apiKey,
		ApiUser:   *apiUser,
		ServerUrl: *serverUrl,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Create Task with name
	create, err := session.Create(
		"Task",
		map[string]interface{}{
			"name": "My Task Name",
		},
	)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Created Task:", create)

	// Delete created Task
	defer func() {
		deleted, err := session.Delete("Task", []string{create.Data["id"].(string)})
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("Task deleted:", deleted.Data)
	}()

	// Update
	update, err := session.Update(
		"Task",
		[]string{create.Data["id"].(string)},
		map[string]interface{}{
			"name": "My Task Name",
		},
	)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Updated Task:", update.Data)
}
