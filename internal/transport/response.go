package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"slices"
	"strings"

	lock "github.com/use-lock/client-go"
)

func Execute[T any](client interface {
	Do(*http.Request) (*http.Response, error)
}, request *http.Request, jsonBody bool, statuses ...int) (*T, error) {
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("lock: read response: %w", err)
	}
	metadata := lock.Response{StatusCode: response.StatusCode, Header: response.Header.Clone(), Body: body}
	if !slices.Contains(statuses, response.StatusCode) {
		apiError := &lock.APIError{}
		_ = json.Unmarshal(body, apiError)
		apiError.Response = metadata
		return nil, apiError
	}
	if !jsonBody {
		result, ok := any(&metadata).(*T)
		if !ok {
			return nil, fmt.Errorf("lock: invalid response type")
		}
		return result, nil
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || (contentType != "application/json" && !strings.HasSuffix(contentType, "+json")) {
		return nil, fmt.Errorf("lock: expected JSON response, received %q", response.Header.Get("Content-Type"))
	}
	var result *T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("lock: decode response: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("lock: expected JSON response, received null")
	}
	return result, nil
}
