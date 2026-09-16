package admin_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/use-lock/client-go/admin"
)

func TestAdminRequestAndPaginatedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/realms" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing bearer token")
		}
		if r.URL.Query().Get("filter[slug]") != "staging,production" || r.URL.Query().Get("page") != "2" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"data":[{"slug":"staging","name":"Staging"}],"links":[],"meta":{"current_page":2,"last_page":3,"total":3}}`)
	}))
	t.Cleanup(server.Close)
	client, err := admin.NewClient(server.URL+"/api", admin.WithToken("test-token"))
	if err != nil {
		t.Fatal(err)
	}
	page := 2
	slugs := []string{"staging", "production"}
	response, err := client.ListRealms(t.Context(), &admin.ListRealmsParams{Page: &page, FilterSlug: &slugs})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Data) != 1 || response.Data[0].Slug != "staging" || response.Meta.CurrentPage != 2 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestAdminErrorResponses(t *testing.T) {
	for _, status := range []int{401, 403, 404} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				io.WriteString(w, `{"error":"invalid_token","error_description":"Denied","message":"Missing realm"}`)
			}))
			t.Cleanup(server.Close)
			client, err := admin.NewClient(server.URL + "/api")
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.GetRealm(t.Context(), "missing")
			var apiError *admin.APIError
			if response != nil || !errors.As(err, &apiError) {
				t.Fatalf("expected APIError, got response=%+v error=%v", response, err)
			}
			if apiError.StatusCode != status || apiError.Code != "invalid_token" || apiError.Description != "Denied" {
				t.Fatalf("unexpected API error: %+v", apiError)
			}
		})
	}
}

func TestCancelledRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("cancelled request reached the server")
	}))
	t.Cleanup(server.Close)
	client, err := admin.NewClient(server.URL + "/api")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = client.GetRealm(ctx, "staging")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestValidationErrorsAndSuccessfulDeletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		io.WriteString(w, `{"message":"Invalid domain","errors":{"domain":["The domain is already taken."]}}`)
	}))
	t.Cleanup(server.Close)
	client, err := admin.NewClient(server.URL + "/api")
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.CreateRealm(t.Context(), admin.CreateRealmData{Name: "Staging", Slug: "staging", Domain: "staging.example"})
	var apiError *admin.APIError
	if created != nil || !errors.As(err, &apiError) || apiError.StatusCode != 422 || len(apiError.Errors["domain"]) != 1 {
		t.Fatalf("unexpected validation error: %v", err)
	}

	deleted, err := client.DeleteRealm(t.Context(), "staging")
	if err != nil {
		t.Fatal(err)
	}
	if deleted.StatusCode != 204 || len(deleted.Body) != 0 {
		t.Fatalf("unexpected deletion response: %+v", deleted)
	}
}

func TestInvalidSuccessResponse(t *testing.T) {
	for _, test := range []struct{ name, contentType, body string }{
		{"invalid JSON", "application/json", "{"},
		{"empty JSON", "application/json", ""},
		{"null JSON", "application/json", "null"},
		{"HTML", "text/html", "<html>Proxy error</html>"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", test.contentType)
				io.WriteString(w, test.body)
			}))
			defer server.Close()
			client, err := admin.NewClient(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			result, err := client.GetRealm(t.Context(), "staging")
			if err == nil || result != nil {
				t.Fatalf("expected decoding error, got %+v, %v", result, err)
			}
		})
	}
}

func TestUnexpectedResponsePreservesStatusHeadersAndBody(t *testing.T) {
	for _, status := range []int{http.StatusBadGateway, http.StatusTooManyRequests, http.StatusAccepted} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "30")
				w.WriteHeader(status)
				io.WriteString(w, "upstream unavailable")
			}))
			defer server.Close()
			client, err := admin.NewClient(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			result, err := client.GetRealm(t.Context(), "staging")
			var apiError *admin.APIError
			if result != nil || !errors.As(err, &apiError) {
				t.Fatalf("expected APIError, got %v", err)
			}
			if apiError.StatusCode != status || apiError.Header.Get("Retry-After") != "30" || string(apiError.Body) != "upstream unavailable" {
				t.Fatalf("response details lost: %+v", apiError)
			}
		})
	}
}

func TestClientRejectsInvalidBaseURL(t *testing.T) {
	for _, baseURL := range []string{"", "/api", "ftp://lock.example", "https://lock.example?token=secret", "https://user:secret@lock.example", "https://lock.example#api"} {
		if _, err := admin.NewClient(baseURL); err == nil {
			t.Errorf("accepted invalid URL %q", baseURL)
		}
	}
}
