package oidc

import (
	"encoding/json"
	"net/url"
	"reflect"
	"testing"
)

func TestFormEncodesResourceArraysAndOptionalValues(t *testing.T) {
	body := json.RawMessage(`{"grant_type":"client_credentials","resource":["https://lock.example/api","https://lock.example/admin-api"],"scope":null,"max_age":0,"include_granted_scopes":false}`)
	got, err := marshalForm(body)
	if err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"grant_type":             {"client_credentials"},
		"resource[]":             {"https://lock.example/api", "https://lock.example/admin-api"},
		"max_age":                {"0"},
		"include_granted_scopes": {"false"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFormRejectsUnsupportedBodies(t *testing.T) {
	for _, body := range []any{json.RawMessage(`{`), json.RawMessage(`[]`), json.RawMessage(`{"resource":{"nested":true}}`)} {
		if _, err := marshalForm(body); err == nil {
			t.Fatalf("expected an error for %s", body)
		}
	}
}
