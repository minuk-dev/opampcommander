// Package formatter provides functions to format data structures into different output formats
// such as YAML, JSON, and text.
package formatter

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"gopkg.in/yaml.v3"
)

// FormatType represents the type of format to be used for output.
type FormatType string

// Format types.
const (
	YAML  FormatType = "yaml"
	JSON  FormatType = "json"
	TEXT  FormatType = "text"
	SHORT FormatType = "short"
)

// Format formats the target as a YAML, JSON, or text representation.
// It's a convenience function that wraps the specific format functions.
func Format(writer io.Writer, target any, t FormatType) error {
	var err error

	switch t {
	case YAML:
		err = FormatYAML(writer, target)
	case JSON:
		err = FormatJSON(writer, target)
	case TEXT:
		err = FormatTEXT(writer, target)
	case SHORT:
		err = FormatShort(writer, target)
	default:
		err = FormatTEXT(writer, target)
	}

	if err != nil {
		return fmt.Errorf("failed to format: %w", err)
	}

	return nil
}

// FormatYAML formats the target as a YAML representation.
func FormatYAML(writer io.Writer, target any) error {
	b, err := yaml.Marshal(target)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml: %w", err)
	}

	_, err = writer.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write yaml: %w", err)
	}

	return nil
}

// FormatJSON formats the target as a JSON representation.
func FormatJSON(writer io.Writer, target any) error {
	b, err := json.MarshalIndent(target, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	_, err = writer.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write json: %w", err)
	}

	return nil
}

// FormatTEXT formats the target as a text representation.
// It uses the field name as the key and the field value as the value.
func FormatTEXT(writer io.Writer, target any) error {
	err := formatStruct(writer, target, false)
	if err != nil {
		return fmt.Errorf("failed to format text: %w", err)
	}

	return nil
}

// FormatShort formats the target struct to a short text representation.
// If the field has a "short" tag, it uses the value of that tag as the key.
func FormatShort(writer io.Writer, target any) error {
	err := formatStruct(writer, target, true)
	if err != nil {
		return fmt.Errorf("failed to format short: %w", err)
	}

	return nil
}

//nolint:errcheck
func formatStruct(writer io.Writer, target any, short bool) error {
	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	if typ.Kind() != reflect.Struct {
		fmt.Fprintf(writer, "%v\n", target)

		return nil
	}

	n := typ.NumField()
	for i := range n {
		field := typ.Field(i)

		value := val.Field(i)
		if !value.CanInterface() {
			continue
		}

		fieldName := field.Name
		shortTag, shortOk := field.Tag.Lookup("short")

		if short && (shortOk && fieldName != "name") { // If field name is "name". Always use it as the key.
			fieldName = shortTag
		}

		if short && (!shortOk && fieldName != "name") { // Name is always used as the key.
			continue // Skip fields without a "short" tag in short mode
		}

		fmt.Fprintf(writer, "%s: ", fieldName)

		if value.Kind() == reflect.Struct {
			fmt.Fprint(writer, "{ ")

			err := formatStruct(writer, value.Interface(), short)
			if err != nil {
				return fmt.Errorf("failed to format struct: %w", err)
			}

			fmt.Fprint(writer, "} ")
		} else {
			fmt.Fprintf(writer, "%v\n", value.Interface())
		}
	}

	return nil
}
