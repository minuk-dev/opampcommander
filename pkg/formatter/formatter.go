package formatter

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"gopkg.in/yaml.v3"
)

type FormatType string

const (
	YAML  FormatType = "yaml"
	JSON  FormatType = "json"
	TEXT  FormatType = "text"
	SHORT FormatType = "short"
)

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

func FormatTEXT(writer io.Writer, target any) error {
	err := formatStruct(writer, target, false)
	if err != nil {
		return fmt.Errorf("failed to format text: %w", err)
	}

	return nil
}

func FormatShort(writer io.Writer, target any) error {
	err := formatStruct(writer, target, true)
	if err != nil {
		return fmt.Errorf("failed to format short: %w", err)
	}

	return nil
}

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
		if shortTag, ok := field.Tag.Lookup("short"); ok && short {
			fieldName = shortTag
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
