// A basic implementation of graceful shutdown using FX dependency-injection
// by using FX.lifecycle hook.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/fx"

	"github.com/koinworks/asgard-heimdal/utils/utinterface"
)

const API_DURATION = 7 * time.Second

func main() {
	app := fx.New(
		provideRedis(),
		provideServer(),
	)

	app.Run()
}

func provideRedis() fx.Option {
	return fx.Options(
		fx.Provide(newRedisConfig),
		fx.Provide(newRedis),
	)
}

type redisConfig struct {
	Address  string
	Password string
}

func newRedisConfig() *redisConfig {
	return &redisConfig{
		Address:  "localhost:6379",
		Password: "",
	}
}

func newRedis(lc fx.Lifecycle, config *redisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Password: config.Password,
	})

	result := client.Ping(context.Background())
	if result.Err() != nil {
		return nil, result.Err()
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			err := client.Close()
			if err != nil {
				return err
			}

			return nil
		},
	})

	return client, nil
}

func provideServer() fx.Option {
	return fx.Options(
		fx.Provide(newServerConfig),
		fx.Provide(newServerMux),
		fx.Invoke(runServer),
	)
}

type serverConfig struct {
	Port string
}

func newServerConfig() (*serverConfig, error) {
	return &serverConfig{
		Port: "8088",
	}, nil
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

func runServer(lc fx.Lifecycle, config *serverConfig, handler http.Handler) error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.Port),
		Handler: handler,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			netListener, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return err
			}

			// run the server using goroutine.
			go func() {
				fmt.Printf("the server started on port %s.\n", config.Port)
				err := server.Serve(netListener)
				if err != nil && err != http.ErrServerClosed {
					log.Println(err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			fmt.Println("tries to shutting down the server...")

			if err := server.Shutdown(ctx); err != nil {
				log.Println(err)
				return err
			}

			fmt.Println("server has been terminated.")
			return nil
		},
	})

	return nil
}
