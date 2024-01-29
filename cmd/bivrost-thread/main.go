// A basic implementation of service that manage multiple threads with
// graceful shutdown using bivrost's thread hook.

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/koinworks/asgard-bivrost/libs"
	"github.com/koinworks/asgard-bivrost/service"
	"github.com/koinworks/asgard-heimdal/constants/cservice"
	"github.com/koinworks/asgard-heimdal/libs/logger"
	"github.com/koinworks/asgard-heimdal/libs/serror"
	"github.com/koinworks/asgard-heimdal/models"

	"github.com/luthfikw/example.graceful-shutdown/internal/bvrouter"
	"github.com/luthfikw/example.graceful-shutdown/internal/component"
	"github.com/luthfikw/example.graceful-shutdown/internal/httprouter"
	"github.com/luthfikw/example.graceful-shutdown/internal/iredis"
)

const API_DURATION = 7 * time.Second

func main() {
	registry, err := libs.InitRegistry(libs.RegistryConfig{
		Service: &models.Service{
			Class:   cservice.ServiceClassUtility,
			Key:     "asgard-tst",
			Name:    "Asgard Tst",
			Version: "v1.0.0",
			Host:    "localhost",
			Port:    4100,
		},
		Address:  "localhost:6379",
		Password: "",
	})
	if err != nil {
		panic(err)
	}

	server, err := libs.NewServer(registry)
	if err != nil {
		panic(err)
	}

	svc := server.AsGatewayService("/test")

	newComponent1(server)
	newComponent2(server)
	newComponent3(server)
	newComponent4(server)

	redisClient, err := newRedis(server)
	if err != nil {
		log.Fatal(err)
	}

	bvrouter.SetupBivrostRouter("0", API_DURATION, svc, redisClient)

	registerServer1(server, redisClient)
	registerServer2(server, redisClient)

	ctx := context.Background()
	err = server.Start(ctx)
	if err != nil {
		panic(err)
	}
}

func newRedis(server *service.Server) (*redis.Client, error) {
	redisClient, err := iredis.NewRedis()
	if err != nil {
		return nil, err
	}

	server.RegisterTrivialTerminationHook("redis client", func(ctx context.Context) {
		if err := redisClient.Close(); err != nil {
			logger.Errf("error while closing the redis client: %+v", err)
		}
	})

	return redisClient, nil
}

func registerServer1(server *service.Server, redisClient *redis.Client) {
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: httprouter.NewHTTPServerMux("1", API_DURATION, redisClient),
	}

	server.RegisterThread("http.server(1)", func(ctx context.Context, terminationCallbackFNChan chan<- func(ctx context.Context)) (errx serror.SError) {
		terminationCallbackFNChan <- func(ctx context.Context) {
			err := httpServer.Shutdown(ctx)
			if err != nil {
				logger.Errf("error while shutdown server #1, details: %+v", err)
			}
		}

		logger.Info("Starting server #1 at port 8080.")
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errx = serror.NewFromErrorc(err, "Failed to start server #1")
			return
		}

		return
	})
}

func registerServer2(server *service.Server, redisClient *redis.Client) {
	httpServer := &http.Server{
		Addr:    ":8081",
		Handler: httprouter.NewHTTPServerMux("2", API_DURATION, redisClient),
	}
	server.RegisterThread("http.server(2)", func(ctx context.Context, terminationCallbackFNChan chan<- func(ctx context.Context)) (errx serror.SError) {
		terminationCallbackFNChan <- func(ctx context.Context) {
			err := httpServer.Shutdown(ctx)
			if err != nil {
				logger.Errf("error while shutdown server #2, details: %+v", err)
			}
		}

		logger.Infof("Starting server #2 at port 8081.")
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errx = serror.NewFromErrorc(err, "Failed to start server #2")
			return
		}

		return
	})
}

func newComponent1(server *service.Server) {
	instance := &component.Component{
		Label: "component-1",
	}
	server.RegisterTrivialTerminationHook(instance.Label, func(ctx context.Context) {
		if err := instance.Dispose(); err != nil {
			logger.Errf("error during disposing '%s': %+v", instance.Label, err)
		}
	})
}

func newComponent2(server *service.Server) {
	instance := &component.Component{
		Label:           "component-2",
		DisposeDuration: time.Second,
	}
	server.RegisterTrivialTerminationHook(instance.Label, func(ctx context.Context) {
		if err := instance.Dispose(); err != nil {
			logger.Errf("error during disposing '%s': %+v", instance.Label, err)
		}
	})
}

func newComponent3(server *service.Server) {
	instance := &component.Component{
		Label:           "component-3",
		DisposeDuration: time.Second * 5,
	}
	server.RegisterTrivialTerminationHook(instance.Label, func(ctx context.Context) {
		if err := instance.Dispose(); err != nil {
			logger.Errf("error during disposing '%s': %+v", instance.Label, err)
		}
	})
}

func newComponent4(server *service.Server) {
	instance := &component.Component{
		Label:           "component-4",
		DisposeDuration: time.Second * 3,
		DisposeError:    errors.New("failed to dispose component-4"),
	}
	server.RegisterTrivialTerminationHook(instance.Label, func(ctx context.Context) {
		if err := instance.Dispose(); err != nil {
			logger.Errf("error during disposing '%s': %+v", instance.Label, err)
		}
	})
}
