// A basic implementation of graceful shutdown using bivrost's
// termination hook.

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/koinworks/asgard-bivrost/libs"
	bvmodels "github.com/koinworks/asgard-bivrost/models"
	"github.com/koinworks/asgard-bivrost/service"
	"github.com/koinworks/asgard-heimdal/constants/cservice"
	"github.com/koinworks/asgard-heimdal/libs/serror"
	"github.com/koinworks/asgard-heimdal/models"
	"github.com/koinworks/asgard-heimdal/utils/utinterface"
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

	redisClient, err := newRedis(server)
	if err != nil {
		log.Fatal(err)
	}

	svc := server.AsGatewayService("/test")
	initServer(svc, redisClient)

	ctx := context.Background()
	err = server.Start(ctx)
	if err != nil {
		panic(err)
	}
}

func newRedis(server *service.Server) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
	})

	result := client.Ping(context.Background())
	if result.Err() != nil {
		return nil, result.Err()
	}

	server.RegisterTrivialTerminationHook("redis.client", func(ctx context.Context) {
		err := client.Close()
		if err != nil {
			log.Println(err)
			return
		}
	})

	return client, nil
}

func initServer(svc *service.Service, redisClient *redis.Client) {
	svc.Get("/", func(ctx *service.Context) service.Result {
		fmt.Println("server got the request...")
		defer func() {
			fmt.Println("server complete the request.")
		}()

		isSlow := utinterface.ToBool(ctx.Query("slow"), false)
		if isSlow {
			time.Sleep(API_DURATION)
		}

		result := redisClient.Get(ctx.Context(), "test")
		if err := result.Err(); err != nil {
			ctx.CaptureSErrors(serror.NewFromError(err))
			return ctx.JSONResponse(500, bvmodels.ResponseBody{
				Message: map[string]string{
					"en": "Failed to get the value from redis",
				},
			})
		}

		return ctx.JSONResponse(200, bvmodels.ResponseBody{
			Message: map[string]string{
				"en": "Success",
			},
			Data: result.Val(),
		})
	})

	svc.Post("/", func(ctx *service.Context) service.Result {
		fmt.Println("server got the request...")
		defer func() {
			fmt.Println("server complete the request.")
		}()

		isSlow := utinterface.ToBool(ctx.Query("slow"), false)
		if isSlow {
			time.Sleep(API_DURATION)
		}

		var payload struct {
			Value string `json:"value"`
		}
		if err := ctx.BodyJSONBind(&payload); err != nil {
			ctx.CaptureSErrors(serror.NewFromError(err))
			return ctx.JSONResponse(400, bvmodels.ResponseBody{
				Message: map[string]string{
					"en": "Request not valid",
				},
			})
		}

		result := redisClient.Set(ctx.Context(), "test", payload.Value, time.Hour)
		if err := result.Err(); err != nil {
			ctx.CaptureSErrors(serror.NewFromError(err))
			return ctx.JSONResponse(500, bvmodels.ResponseBody{
				Message: map[string]string{
					"en": "Failed to store value to redis",
				},
			})
		}

		return ctx.JSONResponse(200, bvmodels.ResponseBody{
			Message: map[string]string{
				"en": "Success",
			},
		})
	})
}
