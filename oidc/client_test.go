package oidc_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/use-lock/client-go/oidc"
)

func TestClientCredentialsFormAndTokenResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth/token" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		for key, want := range map[string]string{"grant_type": "client_credentials", "client_id": "client", "client_secret": "secret+&=", "resource": "https://lock.example/api"} {
			if got := r.PostForm.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"access_token":"token","token_type":"Bearer","expires_in":300}`)
	}))
	t.Cleanup(server.Close)
	client, err := oidc.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	id, secret := "client", "secret+&="
	var resource oidc.OAuthTokenRequest_2_Resource
	if err := resource.FromOAuthTokenRequest2Resource0("https://lock.example/api"); err != nil {
		t.Fatal(err)
	}
	var body oidc.OAuthTokenRequest
	if err := body.FromOAuthTokenRequest2(oidc.OAuthTokenRequest2{GrantType: "client_credentials", ClientID: &id, ClientSecret: &secret, Resource: &resource}); err != nil {
		t.Fatal(err)
	}
	response, err := client.OidcTokenPostWithFormdataBodyWithResponse(t.Context(), body)
	if err != nil {
		t.Fatal(err)
	}
	if response.JSON200 == nil || response.JSON200.AccessToken != "token" {
		t.Fatalf("unexpected response: %+v", response)
	}
}
