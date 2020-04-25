#[Ftrack](https://www.ftrack.com) Golang API

Partially ported from [ftrack-javascript-api](https://bitbucket.org/ftrack/ftrack-javascript-api)

##### Basic usage
```go
package main

import (
    "github.com/conducte/ftrack-golang-api/ftrack"
    "log"
    "os"
)

func main() {
    mustLookupEnv := func(key string) string {
        val, ok := os.LookupEnv(key)
        if !ok {
            log.Fatalf("no key in env: %s", key)
        }
        return val
    }
    session, err := ftrack.NewSession(ftrack.SessionConfig{
        ApiKey:    mustLookupEnv("FTRACK_API_KEY"),
        ApiUser:   mustLookupEnv("FTRACK_API_USER"),
        ServerUrl: mustLookupEnv("FTRACK_SERVER"),
    })
    if err != nil {
        log.Fatal(err)
    }
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
