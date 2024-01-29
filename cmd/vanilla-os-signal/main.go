// A basic implementation of graceful shutdown using listening os-signal.

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/luthfikw/example.graceful-shutdown/internal/component"
	"github.com/luthfikw/example.graceful-shutdown/internal/httprouter"
	"github.com/luthfikw/example.graceful-shutdown/internal/iredis"
)

const API_DURATION = 7 * time.Second

func main() {
	// load components.
	component1 := newComponent1()
	component2 := newComponent2()
	component3 := newComponent3()
	component4 := newComponent4()

	redisClient, err := iredis.NewRedis()
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

		if err := component1.Dispose(); err != nil {
			fmt.Printf("error during disposing '%s': %+v\n", component1.Label, err)
		}

		if err := component2.Dispose(); err != nil {
			fmt.Printf("error during disposing '%s': %+v\n", component2.Label, err)
		}

		if err := component3.Dispose(); err != nil {
			fmt.Printf("error during disposing '%s': %+v\n", component3.Label, err)
		}

		if err := component4.Dispose(); err != nil {
			fmt.Printf("error during disposing '%s': %+v\n", component4.Label, err)
		}

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

func newServer(redisClient *redis.Client) (*http.Server, error) {
	server := &http.Server{
		Addr:    ":8088",
		Handler: httprouter.NewHTTPServerMux("0", API_DURATION, redisClient),
	}
	return server, nil
}

func newComponent1() *component.Component {
	return &component.Component{
		Label: "component-1",
	}
}

func newComponent2() *component.Component {
	return &component.Component{
		Label:           "component-2",
		DisposeDuration: time.Second,
	}
}

func newComponent3() *component.Component {
	return &component.Component{
		Label:           "component-3",
		DisposeDuration: time.Second * 5,
	}
}

func newComponent4() *component.Component {
	return &component.Component{
		Label:           "component-4",
		DisposeDuration: time.Second * 3,
		DisposeError:    errors.New("failed to dispose component-4"),
	}
}
