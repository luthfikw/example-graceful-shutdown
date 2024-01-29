package httprouter

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/koinworks/asgard-heimdal/utils/utinterface"
)

func NewHTTPServerMux(label string, apiDuration time.Duration, redisClient *redis.Client) http.Handler {
	var serverMux http.ServeMux
	serverMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("server '%s' got the request...\n", label)
		defer func() {
			fmt.Printf("server '%s' complete the request.\n", label)
		}()

		isSlow := utinterface.ToBool(r.URL.Query().Get("slow"), false)
		if isSlow {
			time.Sleep(apiDuration)
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

	return &serverMux
}
