// A basic implementation of graceful shutdown using listening os-signal.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/koinworks/asgard-heimdal/utils/utinterface"
)

const API_DURATION = 7 * time.Second

func main() {
	redisClient, err := newRedis()
	if err != nil {
		log.Fatal(err)
	}

	server, err := newServer(redisClient)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup

	// 1. os signal listener.
	// 2. running the server.
	wg.Add(2)

	go func() {
		osSignal := make(chan os.Signal, 1)
		signal.Notify(osSignal,
			os.Interrupt,
			os.Kill,
			syscall.SIGINT,
			syscall.SIGKILL,
			syscall.SIGTERM,
			syscall.SIGHUP,
			syscall.SIGQUIT,
		)

		<-osSignal

		fmt.Println("terminating the server...")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Println(err)
		}
		fmt.Println("server has been terminated.")

		fmt.Println("closing the redis client...")
		if err := redisClient.Close(); err != nil {
			log.Println(err)
		}
		fmt.Println("redis has been closed.")

		wg.Done()
	}()

	go func() {
		fmt.Println("the server started on port 8088.")
		err = server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}

		wg.Done()
	}()

	wg.Wait()
}

func newRedis() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
	})

	result := client.Ping(context.Background())
	if result.Err() != nil {
		return nil, result.Err()
	}

	return client, nil
}

func newServer(redisClient *redis.Client) (*http.Server, error) {
	mux, err := newServerMux(redisClient)
	if err != nil {
		return nil, err
	}

	server := &http.Server{
		Addr:    ":8088",
		Handler: mux,
	}
	return server, nil
}

func newServerMux(redisClient *redis.Client) (http.Handler, error) {
	var mux http.ServeMux
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("server got the request...\n")
		defer func() {
			fmt.Printf("server complete the request.\n")
		}()

		isSlow := utinterface.ToBool(r.URL.Query().Get("slow"), false)
		if isSlow {
			time.Sleep(API_DURATION)
		}

		switch r.Method {
		case "GET":
			result := redisClient.Get(r.Context(), "test")
			if err := result.Err(); err != nil {
				log.Println(err)
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(200)
			fmt.Fprintf(w, "Value: %s", result.Val())

		case "POST":
			var payload struct {
				Value string `json:"value"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				w.WriteHeader(400)
				return
			}

			result := redisClient.Set(r.Context(), "test", payload.Value, time.Hour)
			if err := result.Err(); err != nil {
				log.Println(err)
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(200)

		default:
			w.WriteHeader(404)
		}
	})

	return &mux, nil
}
