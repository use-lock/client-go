package management_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/use-lock/client-go/management"
)

func TestPatchPreservesOmittedNullAndZeroValues(t *testing.T) {
	consent := false
	redirects := []string{}
	for _, test := range []struct {
		name string
		body management.UpdateClientData
		want string
	}{
		{"omitted", management.UpdateClientData{}, `{}`},
		{"null", management.UpdateClientData{BackchannelLogoutURI: nullable.NewNullNullable[string]()}, `{"backchannel_logout_uri":null}`},
		{"value", management.UpdateClientData{BackchannelLogoutURI: nullable.NewNullableWithValue("https://app.example/logout")}, `{"backchannel_logout_uri":"https://app.example/logout"}`},
		{"zero values", management.UpdateClientData{ConsentRequired: &consent, RedirectUris: &redirects}, `{"consent_required":false,"redirect_uris":[]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			request, err := management.NewPatchClientRequest("https://lock.example/api/", "staging", "client-123", test.body)
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
