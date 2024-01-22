// A basic implementation of service that manage multiple threads.

package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		spawnServer1()
		wg.Done()
	}()

	go func() {
		spawnServer2()
		wg.Done()
	}()

	wg.Wait()
}

func spawnServer1() {
	fmt.Printf("Starting server #1 at port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Println(err)
	}
}

func spawnServer2() {
	fmt.Printf("Starting server #2 at port 8081\n")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Println(err)
	}
}
