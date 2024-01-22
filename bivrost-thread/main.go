// A basic implementation of service that manage multiple threads with
// graceful shutdown using bivrost's thread hook.

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

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

	svc := server.AsGatewayService("/test")
	initServer0(svc)

	registerServer1(server)
	registerServer2(server)

	ctx := context.Background()
	err = server.Start(ctx)
	if err != nil {
		panic(err)
	}
}

func initServer0(svc *service.Service) {
	svc.Get("/", func(ctx *service.Context) service.Result {
		fmt.Println("server 0 got the request...")
		defer func() {
			fmt.Println("server 0 complete the request.")
		}()

		isSlow := utinterface.ToBool(ctx.Query("slow"), false)
		if isSlow {
			time.Sleep(API_DURATION)
		}

		return ctx.JSONResponse(200, bvmodels.ResponseBody{
			Message: map[string]string{
				"en": "Hello world!",
				"id": "Halo dunia!",
			},
		})
	})
}

func registerServer1(server *service.Server) {
	httpServer := &http.Server{
		Addr: ":8080",
	}
	initServerRoute("1", httpServer)

	server.RegisterThread("http.server(1)", func(ctx context.Context, terminationCallbackFNChan chan<- func(ctx context.Context)) (errx serror.SError) {
		terminationCallbackFNChan <- func(ctx context.Context) {
			err := httpServer.Shutdown(ctx)
			if err != nil {
				fmt.Printf("error while shutdown server #1, details: %+v", err)
			}
		}

		fmt.Printf("Starting server #1 at port 8080\n")
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errx = serror.NewFromErrorc(err, "Failed to start server #1")
			return
		}

		return
	})
}

func registerServer2(server *service.Server) {
	httpServer := &http.Server{
		Addr: ":8081",
	}
	initServerRoute("2", httpServer)

	server.RegisterThread("http.server(2)", func(ctx context.Context, terminationCallbackFNChan chan<- func(ctx context.Context)) (errx serror.SError) {
		terminationCallbackFNChan <- func(ctx context.Context) {
			err := httpServer.Shutdown(ctx)
			if err != nil {
				fmt.Printf("error while shutdown server #2, details: %+v", err)
			}
		}

		fmt.Printf("Starting server #2 at port 8081\n")
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errx = serror.NewFromErrorc(err, "Failed to start server #2")
			return
		}

		return
	})
}

func initServerRoute(name string, server *http.Server) {
	var mux http.ServeMux
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("server %s got the request...\n", name)
		defer func() {
			fmt.Printf("server %s complete the request.\n", name)
		}()

		isSlow := utinterface.ToBool(r.URL.Query().Get("slow"), false)
		if isSlow {
			time.Sleep(API_DURATION)
		}

		fmt.Fprintln(w, "Hello world!")
		w.WriteHeader(200)
	})

	server.Handler = &mux
}
