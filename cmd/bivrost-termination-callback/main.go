// A basic implementation of graceful shutdown using bivrost's
// termination hook.

package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/koinworks/asgard-bivrost/libs"
	"github.com/koinworks/asgard-bivrost/service"
	"github.com/koinworks/asgard-heimdal/constants/cservice"
	"github.com/koinworks/asgard-heimdal/libs/logger"
	"github.com/koinworks/asgard-heimdal/models"

	"github.com/luthfikw/example.graceful-shutdown/internal/bvrouter"
	"github.com/luthfikw/example.graceful-shutdown/internal/component"
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

	newComponent1(server)
	newComponent2(server)
	newComponent3(server)
	newComponent4(server)

	redisClient, err := newRedis(server)
	if err != nil {
		log.Fatal(err)
	}

	svc := server.AsGatewayService("/test")
	bvrouter.SetupBivrostRouter("0", API_DURATION, svc, redisClient)

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

	server.RegisterTrivialTerminationHook("redis.client", func(ctx context.Context) {
		err := redisClient.Close()
		if err != nil {
			logger.Errf("error during closing the redis client: %+v", err)
			return
		}
	})

	return redisClient, nil
}

func newComponent1(server *service.Server) {
	instance := &component.Component{
		Label: "component-1",
	}
	server.RegisterTrivialTerminationHook(instance.Label, func(ctx context.Context) {
		if err := instance.Dispose(); err != nil {
			logger.Errf("error during disposing '%s': %+v\n", instance.Label, err)
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
			logger.Errf("error during disposing '%s': %+v\n", instance.Label, err)
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
			logger.Errf("error during disposing '%s': %+v\n", instance.Label, err)
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
			logger.Errf("error during disposing '%s': %+v\n", instance.Label, err)
		}
	})
}
