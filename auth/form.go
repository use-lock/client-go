package auth

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// Generated oneOf types keep their fields in an unexported JSON union, which
// runtime.MarshalForm cannot see. Encode their JSON representation instead.
func marshalForm(body any) (url.Values, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	values := make(url.Values)
	for name, raw := range fields {
		if string(raw) == "null" {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) == nil {
			values.Set(name, value)
			continue
		}
		var items []string
		if json.Unmarshal(raw, &items) == nil {
			for _, item := range items {
				values.Add(name+"[]", item)
			}
			continue
		}
		var scalar any
		if err := json.Unmarshal(raw, &scalar); err != nil {
			return nil, err
		}
		switch scalar.(type) {
		case bool, float64:
			values.Set(name, string(raw))
		default:
			return nil, fmt.Errorf("unsupported form field %q", name)
		}
	}
	return values, nil
}
