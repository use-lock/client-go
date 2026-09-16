package lock

import (
	"fmt"
	"net/http"
)

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

type APIError struct {
	Response
	Code        string              `json:"error"`
	Message     string              `json:"message"`
	Description string              `json:"error_description"`
	Errors      map[string][]string `json:"errors"`
}

func (e *APIError) Error() string {
	message := e.Description
	if message == "" {
		message = e.Message
	}
	if message == "" {
		message = e.Code
	}
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	return fmt.Sprintf("lock: HTTP %d: %s", e.StatusCode, message)
}
