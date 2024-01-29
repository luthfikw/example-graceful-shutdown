// A basic implementation of service that manage multiple threads.

package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/luthfikw/example.graceful-shutdown/internal/httprouter"
	"github.com/luthfikw/example.graceful-shutdown/internal/iredis"
)

const API_DURATION = 7 * time.Second

func main() {
	redisClient, err := iredis.NewRedis()
	if err != nil {
		log.Fatal(err)
	}

	httpHandler := httprouter.NewHTTPServerMux("0", API_DURATION, redisClient)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		spawnServer1(httpHandler)
		wg.Done()
	}()

	go func() {
		spawnServer2(httpHandler)
		wg.Done()
	}()

	wg.Wait()
}

func spawnServer1(httpHandler http.Handler) {
	fmt.Printf("Starting server #1 at port 8080\n")
	if err := http.ListenAndServe(":8080", httpHandler); err != nil {
		log.Println(err)
	}
}

func spawnServer2(httpHandler http.Handler) {
	fmt.Printf("Starting server #2 at port 8081\n")
	if err := http.ListenAndServe(":8081", httpHandler); err != nil {
		log.Println(err)
	}
}
