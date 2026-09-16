package auth_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/use-lock/client-go/auth"
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
	client, err := auth.NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	id, secret := "client", "secret+&="
	var resource auth.ClientCredentialsRequest_Resource
	if err := resource.FromClientCredentialsRequestResource0("https://lock.example/api"); err != nil {
		t.Fatal(err)
	}
	var body auth.OAuthTokenRequest
	if err := body.FromClientCredentialsRequest(auth.ClientCredentialsRequest{GrantType: "client_credentials", ClientID: &id, ClientSecret: &secret, Resource: &resource}); err != nil {
		t.Fatal(err)
	}
	response, err := client.IssueToken(t.Context(), body)
	if err != nil {
		t.Fatal(err)
	}
	if response.AccessToken != "token" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestAuthorizationRedirectIsReturnedWithoutFollowingIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/authorize" {
			t.Errorf("redirect was followed: %s", r.URL.Path)
		}
		w.Header().Set("Location", "/auth/login")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()
	client, err := auth.NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Authorize(t.Context(), &auth.AuthorizeParams{})
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusFound || response.Header.Get("Location") != "/auth/login" {
		t.Fatalf("unexpected redirect: %+v", response)
	}
}
