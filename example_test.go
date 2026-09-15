package lock_test

import (
	"context"
	"log"
	"net/http"
	"time"

	lock "github.com/use-lock/client-go"
)

func ExampleNewClientWithResponses() {
	client, err := lock.NewClientWithResponses("https://lock.example/api",
		lock.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
		lock.WithRequestEditorFn(func(_ context.Context, request *http.Request) error {
			request.Header.Set("Authorization", "Bearer <access-token>")
			return nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	response, err := client.V1RealmsShowWithResponse(context.Background(), "staging")
	if err != nil {
		log.Fatal(err)
	}
	if response.JSON200 == nil {
		log.Fatalf("Lock API returned HTTP %d", response.StatusCode())
	}
	log.Print(response.JSON200.Data.Name)
}
