// Package yamlfile provides a helper for reading a YAML resource definition
// and unmarshaling it into a target struct via the struct's `json` tags.
//
// The v1 API types are tagged with `json` only (no `yaml` tags); yaml.v3 does
// not honor `json` tags. To bridge the two, the YAML is first decoded into a
// generic value, re-encoded as JSON, and finally decoded into the target.
package yamlfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load reads the YAML file at path and unmarshals it into target.
func Load(path string, target any) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	return Unmarshal(data, target)
}

// Unmarshal parses YAML bytes and routes through JSON so json struct tags
// on the target are honored.
func Unmarshal(data []byte, target any) error {
	var generic any

	err := yaml.Unmarshal(data, &generic)
	if err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}

	jsonBytes, err := json.Marshal(generic)
	if err != nil {
		return fmt.Errorf("re-encode as json: %w", err)
	}

	err = json.Unmarshal(jsonBytes, target)
	if err != nil {
		return fmt.Errorf("unmarshal into target: %w", err)
	}

	return nil
}
