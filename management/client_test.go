package management_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/use-lock/client-go/management"
)

func TestPatchPreservesOmittedNullAndZeroValues(t *testing.T) {
	consent := false
	redirects := []string{}
	audiences := []string{"https://lock.example/docpipe"}
	for _, test := range []struct {
		name string
		body management.UpdateClientData
		want string
	}{
		{"omitted", management.UpdateClientData{}, `{}`},
		{"null", management.UpdateClientData{BackchannelLogoutURI: nullable.NewNullNullable[string]()}, `{"backchannel_logout_uri":null}`},
		{"value", management.UpdateClientData{BackchannelLogoutURI: nullable.NewNullableWithValue("https://app.example/logout")}, `{"backchannel_logout_uri":"https://app.example/logout"}`},
		{"zero values", management.UpdateClientData{ConsentRequired: &consent, RedirectUris: &redirects}, `{"consent_required":false,"redirect_uris":[]}`},
		{"allowed audiences", management.UpdateClientData{AllowedAudiences: &audiences}, `{"allowed_audiences":["https://lock.example/docpipe"]}`},
		{"cleared allowed audiences", management.UpdateClientData{AllowedAudiences: &redirects}, `{"allowed_audiences":[]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			request, err := management.NewUpdateClientRequest("https://lock.example/api/", "staging", "client-123", test.body)
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

func TestCreateClientSendsAndReadsAllowedAudiences(t *testing.T) {
	audiences := []string{"https://lock.example/docpipe", "https://docpipe.example"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(body["allowed_audiences"], []any{audiences[0], audiences[1]}) {
			t.Errorf("unexpected allowed_audiences: %#v", body["allowed_audiences"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"data":{"client":{"client_id":"docpipe","allowed_audiences":["https://lock.example/docpipe","https://docpipe.example"]},"credentials":{"client_id":"docpipe","secret":"s3cret"}}}`)
	}))
	t.Cleanup(server.Close)
	client, err := management.NewClient(server.URL + "/api")
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.CreateClient(t.Context(), "staging", management.CreateClientData{
		Name:                    "Docpipe",
		TokenEndpointAuthMethod: management.ClientSecretBasic,
		GrantTypes:              []management.CreateClientDataGrantTypes{management.CreateClientDataGrantTypesClientCredentials},
		AllowedAudiences:        &audiences,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(response.Data.Client.AllowedAudiences, audiences) {
		t.Fatalf("unexpected allowed audiences: %#v", response.Data.Client.AllowedAudiences)
	}
}
