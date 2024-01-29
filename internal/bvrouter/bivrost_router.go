package bvrouter

import (
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	bvmodels "github.com/koinworks/asgard-bivrost/models"
	"github.com/koinworks/asgard-bivrost/service"
	"github.com/koinworks/asgard-heimdal/libs/serror"
	"github.com/koinworks/asgard-heimdal/utils/utinterface"
)

var (
	successMessage = map[string]string{
		"en": "Success",
		"id": "Berhasil",
	}
	failedMessage = map[string]string{
		"en": "Request is not valid",
		"id": "Permintaan tidak valid",
	}
	errorMessage = map[string]string{
		"en": "Oops! Something went wrong, please try again later",
		"id": "Ups! Ada yang tidak beres, coba lagi nanti",
	}
)

func SetupBivrostRouter(label string, apiDuration time.Duration, svc *service.Service, redisClient *redis.Client) {
	svc.Get("/", func(ctx *service.Context) service.Result {
		fmt.Printf("server '%s' got the request...\n", label)
		defer func() {
			fmt.Printf("server '%s' complete the request.\n", label)
		}()

		isSlow := utinterface.ToBool(ctx.Query("slow"), false)
		if isSlow {
			time.Sleep(apiDuration)
		}

		result := redisClient.Get(ctx.Context(), "test")
		if err := result.Err(); err != nil {
			ctx.CaptureSErrors(serror.NewFromErrorc(err, "failed to get data from redis"))
			return ctx.JSONResponse(500, bvmodels.ResponseBody{
				Message: errorMessage,
			})
		}

		return ctx.JSONResponse(200, bvmodels.ResponseBody{
			Message: successMessage,
			Data:    result.Val(),
		})
	})

	svc.Post("/", func(ctx *service.Context) service.Result {
		var payload struct {
			Value string `json:"value"`
		}
		if err := ctx.BodyJSONBind(&payload); err != nil {
			ctx.CaptureSErrors(serror.NewFromErrorc(err, "failed to read the request payload"))
			return ctx.JSONResponse(400, bvmodels.ResponseBody{
				Message: failedMessage,
			})
		}

		isSlow := utinterface.ToBool(ctx.Query("slow"), false)
		if isSlow {
			time.Sleep(apiDuration)
		}

		result := redisClient.Set(ctx.Context(), "test", payload.Value, time.Hour)
		if err := result.Err(); err != nil {
			ctx.CaptureSErrors(serror.NewFromErrorc(err, "failed to write data to redis"))
			return ctx.JSONResponse(500, bvmodels.ResponseBody{
				Message: errorMessage,
			})
		}

		return ctx.JSONResponse(200, bvmodels.ResponseBody{
			Message: successMessage,
		})
	})
}
