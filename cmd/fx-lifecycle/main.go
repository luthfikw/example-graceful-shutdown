// A basic implementation of graceful shutdown using FX dependency-injection
// by using FX.lifecycle hook.

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/fx"

	"github.com/luthfikw/example.graceful-shutdown/internal/component"
	"github.com/luthfikw/example.graceful-shutdown/internal/httprouter"
)

const API_DURATION = 7 * time.Second

func main() {
	app := fx.New(
		fx.Invoke(newComponent1),
		fx.Invoke(newComponent2),
		fx.Invoke(newComponent3),
		fx.Invoke(newComponent4),

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
	httpHandler := httprouter.NewHTTPServerMux("0", API_DURATION, redisClient)
	return httpHandler, nil
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

func newComponent1(lc fx.Lifecycle) {
	instance := &component.Component{
		Label: "component-1",
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return instance.Dispose()
		},
	})
}

func newComponent2(lc fx.Lifecycle) {
	instance := &component.Component{
		Label:           "component-2",
		DisposeDuration: time.Second,
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return instance.Dispose()
		},
	})
}

func newComponent3(lc fx.Lifecycle) {
	instance := &component.Component{
		Label:           "component-3",
		DisposeDuration: time.Second * 5,
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return instance.Dispose()
		},
	})
}

func newComponent4(lc fx.Lifecycle) {
	instance := &component.Component{
		Label:           "component-4",
		DisposeDuration: time.Second * 3,
		DisposeError:    errors.New("failed to dispose component-4"),
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return instance.Dispose()
		},
	})
}
