// Package formatter provides functions to format data structures into different output formats
// such as YAML, JSON, and text.
package formatter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"text/tabwriter"

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
		err = FormatText(writer, target)
	case SHORT:
		err = FormatShort(writer, target)
	default:
		err = FormatText(writer, target)
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

// FormatText formats the target as a text representation.
// It uses the field name as the key and the field value as the value.
//
//nolint:varnamelen,err113,intrange,errcheck,gosec,mnd,funlen,exhaustive
func FormatText(w io.Writer, data any) error {
	v := reflect.ValueOf(data)

	var slice reflect.Value

	switch v.Kind() {
	case reflect.Slice:
		if v.Len() == 0 {
			return errors.New("no data to display")
		}

		slice = v
	case reflect.Struct, reflect.Ptr:
		slice = reflect.MakeSlice(reflect.SliceOf(v.Type()), 1, 1)
		slice.Index(0).Set(v)
	default:
		return errors.New("data must be a slice or struct")
	}

	firstElem := slice.Index(0)
	if firstElem.Kind() == reflect.Ptr {
		firstElem = firstElem.Elem()
	}

	elemType := firstElem.Type()

	var (
		fieldIndexes []int
		fieldNames   []string
	)

	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		if field.Name == "Name" || field.Name == "ID" || field.Tag.Get("text") != "" {
			fieldIndexes = append(fieldIndexes, i)
			fieldNames = append(fieldNames, field.Name)
		}
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Header
	for _, name := range fieldNames {
		fmt.Fprintf(tw, "%s\t", name)
	}

	fmt.Fprintln(tw)

	// Separator
	for range fieldNames {
		fmt.Fprintf(tw, "--------\t")
	}

	fmt.Fprintln(tw)

	// Rows
	for i := 0; i < slice.Len(); i++ {
		row := slice.Index(i)
		if row.Kind() == reflect.Ptr {
			row = row.Elem()
		}

		for _, idx := range fieldIndexes {
			fmt.Fprintf(tw, "%v\t", row.Field(idx).Interface())
		}

		fmt.Fprintln(tw)
	}

	tw.Flush()

	return nil
}

// FormatShort formats the target struct to a short text representation.
// If the field has a "short" tag, it uses the value of that tag as the key.
//
//nolint:errcheck,varnamelen,mnd,intrange,gosec,funlen,err113,exhaustive
func FormatShort(w io.Writer, data any) error {
	v := reflect.ValueOf(data)

	var slice reflect.Value

	switch v.Kind() {
	case reflect.Slice:
		if v.Len() == 0 {
			return errors.New("no data to display")
		}

		slice = v
	case reflect.Struct, reflect.Ptr:
		slice = reflect.MakeSlice(reflect.SliceOf(v.Type()), 1, 1)
		slice.Index(0).Set(v)
	default:
		return errors.New("data must be a slice or struct")
	}

	firstElem := slice.Index(0)
	if firstElem.Kind() == reflect.Ptr {
		firstElem = firstElem.Elem()
	}

	elemType := firstElem.Type()

	var (
		fieldIndexes []int
		fieldNames   []string
	)

	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		if field.Name == "Name" || field.Name == "ID" || field.Tag.Get("short") != "" {
			fieldIndexes = append(fieldIndexes, i)
			fieldNames = append(fieldNames, field.Name)
		}
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Header
	for _, name := range fieldNames {
		fmt.Fprintf(tw, "%s\t", name)
	}

	fmt.Fprintln(tw)

	// Separator
	for range fieldNames {
		fmt.Fprintf(tw, "--------\t")
	}

	fmt.Fprintln(tw)

	// Rows
	for i := 0; i < slice.Len(); i++ {
		row := slice.Index(i)
		if row.Kind() == reflect.Ptr {
			row = row.Elem()
		}

		for _, idx := range fieldIndexes {
			fmt.Fprintf(tw, "%v\t", row.Field(idx).Interface())
		}

		fmt.Fprintln(tw)
	}

	tw.Flush()

	return nil
}
