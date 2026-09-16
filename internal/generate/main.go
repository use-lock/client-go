package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type object = map[string]any

func main() {
	if err := generate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func generate() error {
	var names map[string]string
	if err := readJSON("openapi/operations.json", &names); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp("", "lock-client-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	for _, group := range []string{"admin", "auth", "management"} {
		var document object
		if err := readJSON("openapi/"+group+".json", &document); err != nil {
			return err
		}
		if err := prepare(document, names); err != nil {
			return fmt.Errorf("%s: %w", group, err)
		}
		data, err := json.MarshalIndent(document, "", "  ")
		if err != nil {
			return err
		}
		spec := filepath.Join(temporary, group+".json")
		if err := os.WriteFile(spec, data, 0600); err != nil {
			return err
		}
		command := exec.Command("go", "tool", "oapi-codegen", "--config=openapi/"+group+".yaml", spec)
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		if err := command.Run(); err != nil {
			return err
		}
	}
	return nil
}

func prepare(document object, names map[string]string) error {
	components := document["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	seen := map[string]bool{}
	for _, item := range document["paths"].(map[string]any) {
		for method, value := range item.(map[string]any) {
			if !strings.Contains(" get post put patch delete head options trace ", " "+method+" ") {
				continue
			}
			operation := value.(map[string]any)
			id := operation["operationId"].(string)
			name, ok := names[id]
			if !ok {
				return fmt.Errorf("missing public operation name for %s", id)
			}
			if seen[name] {
				return fmt.Errorf("duplicate public operation name %s", name)
			}
			seen[name] = true
			operation["operationId"] = name
			result := "Response"
			jsonResponse := true
			statuses := []string{}
			for status, value := range operation["responses"].(map[string]any) {
				code, err := strconv.Atoi(status)
				if err != nil || code < 200 || code >= 400 {
					continue
				}
				statuses = append(statuses, status)
				response := value.(map[string]any)
				content, _ := response["content"].(map[string]any)
				if code >= 300 || content["text/html"] != nil {
					jsonResponse = false
					continue
				}
				media, ok := content["application/json"].(map[string]any)
				if !ok {
					continue
				}
				schema := media["schema"].(map[string]any)
				if ref, ok := schema["$ref"].(string); ok {
					result = strings.TrimPrefix(ref, "#/components/schemas/")
				} else {
					result = name + "Result"
					props, _ := schema["properties"].(map[string]any)
					if data, ok := props["data"].(map[string]any); ok {
						if ref, ok := data["$ref"].(string); ok {
							result = strings.TrimSuffix(filepath.Base(ref), "Data") + "Response"
						}
						if items, ok := data["items"].(map[string]any); ok {
							if ref, ok := items["$ref"].(string); ok {
								result = strings.TrimSuffix(filepath.Base(ref), "Data") + "Collection"
							}
						}
					}
					if existing, ok := schemas[result]; ok {
						a, _ := json.Marshal(existing)
						b, _ := json.Marshal(schema)
						if string(a) != string(b) {
							return fmt.Errorf("conflicting response schema %s", result)
						}
					}
					schemas[result] = schema
					media["schema"] = object{"$ref": "#/components/schemas/" + result}
				}
			}
			if len(statuses) == 0 {
				return fmt.Errorf("no success response for %s", name)
			}
			if !jsonResponse || result == "Response" {
				jsonResponse = false
				result = "Response"
			}
			sort.Strings(statuses)
			operation["x-client-result"] = result
			operation["x-client-json"] = jsonResponse
			operation["x-client-statuses"] = strings.Join(statuses, ", ")
		}
	}
	return nil
}
