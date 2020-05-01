# [Ftrack](https://www.ftrack.com) Golang API

Partially ported from [ftrack-javascript-api](https://bitbucket.org/ftrack/ftrack-javascript-api)

##### Basic usage
```go
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

```

#### Roadmap:

- Documentation and examples
- [EventHub](https://bitbucket.org/ftrack/ftrack-javascript-api/src/master/source/event_hub.js) support
- Entity type, to allow easy entity manipulations
- More tests

Contributions and issues welcomed as well as any feedback!  
