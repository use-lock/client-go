package admin_test

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/use-lock/client-go/admin"
)

func ExampleNewClient() {
	client, err := admin.NewClient("https://lock.example/api",
		admin.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
		admin.WithToken("<access-token>"),
	)
	if err != nil {
		log.Fatal(err)
	}
	response, err := client.GetRealm(context.Background(), "staging")
	if err != nil {
		log.Fatal(err)
	}
	log.Print(response.Data.Name)
}
