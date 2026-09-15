package lock_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/oapi-codegen/nullable"
	lock "github.com/use-lock/client-go"
)

func TestManagementRequestAndPaginatedResponse(t *testing.T) {
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
	client, err := lock.NewClientWithResponses(server.URL+"/api", lock.WithRequestEditorFn(func(_ context.Context, r *http.Request) error {
		r.Header.Set("Authorization", "Bearer test-token")
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	page := 2
	slugs := []string{"staging", "production"}
	response, err := client.V1RealmsIndexWithResponse(t.Context(), &lock.V1RealmsIndexParams{Page: &page, FilterSlug: &slugs})
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil || len(response.JSON200.Data) != 1 || response.JSON200.Data[0].Slug != "staging" || response.JSON200.Meta.CurrentPage != 2 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestPatchPreservesOmittedNullAndZeroValues(t *testing.T) {
	consent := false
	redirects := []string{}
	for _, test := range []struct {
		name string
		body lock.UpdateClientData
		want string
	}{
		{"omitted", lock.UpdateClientData{}, `{}`},
		{"null", lock.UpdateClientData{BackchannelLogoutURI: nullable.NewNullNullable[string]()}, `{"backchannel_logout_uri":null}`},
		{"value", lock.UpdateClientData{BackchannelLogoutURI: nullable.NewNullableWithValue("https://app.example/logout")}, `{"backchannel_logout_uri":"https://app.example/logout"}`},
		{"zero values", lock.UpdateClientData{ConsentRequired: &consent, RedirectUris: &redirects}, `{"consent_required":false,"redirect_uris":[]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			request, err := lock.NewAPIV1RealmsClientsUpdatePatchRequest("https://lock.example/api/", "staging", "client-123", test.body)
			if err != nil {
				t.Fatal(err)
			}
			defer request.Body.Close()
			if request.Method != http.MethodPatch || request.URL.Path != "/api/v1/realms/staging/clients/client-123" {
				t.Fatalf("unexpected request: %s %s", request.Method, request.URL)
			}
			var got, want map[string]any
			if err := json.NewDecoder(request.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}
		})
	}
}

func TestManagementErrorResponses(t *testing.T) {
	for _, status := range []int{401, 403, 404} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				io.WriteString(w, `{"error":"invalid_token","error_description":"Denied","message":"Missing realm"}`)
			}))
			t.Cleanup(server.Close)
			client, err := lock.NewClientWithResponses(server.URL + "/api")
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.V1RealmsShowWithResponse(t.Context(), "missing")
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode() != status || response.JSON200 != nil {
				t.Fatalf("unexpected response: %+v", response)
			}
			if (status == 401 && response.JSON401 == nil) || (status == 403 && response.JSON403 == nil) || (status == 404 && response.JSON404 == nil) {
				t.Fatal("error body was not decoded")
			}
		})
	}
}

func TestCancelledRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("cancelled request reached the server")
	}))
	t.Cleanup(server.Close)
	client, err := lock.NewClientWithResponses(server.URL + "/api")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = client.V1RealmsShowWithResponse(ctx, "staging")
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
	client, err := lock.NewClientWithResponses(server.URL + "/api")
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.V1RealmsStoreWithResponse(t.Context(), lock.CreateRealmData{Name: "Staging", Slug: "staging", Domain: "staging.example"})
	if err != nil {
		t.Fatal(err)
	}
	if created.StatusCode() != 422 || created.JSON422 == nil || len(created.JSON422.Errors["domain"]) != 1 {
		t.Fatalf("unexpected validation response: %+v", created)
	}
	deleted, err := client.V1RealmsDestroyWithResponse(t.Context(), "staging")
	if err != nil {
		t.Fatal(err)
	}
	if deleted.StatusCode() != 204 || len(deleted.Body) != 0 {
		t.Fatalf("unexpected deletion response: %+v", deleted)
	}
}
