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

func Format(w io.Writer, target any, t FormatType) {
	switch t {
	case YAML:
		FormatYAML(w, target)
	case JSON:
		FormatJSON(w, target)
	case TEXT:
		FormatTEXT(w, target)
	case SHORT:
		FormatSHORT(w, target)
	default:
		FormatTEXT(w, target)
	}
}

func FormatYAML(w io.Writer, target any) {
	b, err := yaml.Marshal(target)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		return
	}
	w.Write(b)
}

func FormatJSON(w io.Writer, target any) {
	b, err := json.MarshalIndent(target, "", "  ")
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		return
	}
	w.Write(b)
}

func FormatTEXT(w io.Writer, target any) {
	formatStruct(w, target, false)
}

func FormatSHORT(w io.Writer, target any) {
	formatStruct(w, target, true)
}

func formatStruct(w io.Writer, target any, short bool) {
	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()
	if typ.Kind() != reflect.Struct {
		fmt.Fprintf(w, "%v\n", target)
		return
	}

	n := typ.NumField()
	for i := 0; i < n; i++ {
		field := typ.Field(i)
		value := val.Field(i)
		if !value.CanInterface() {
			continue
		}

		fieldName := field.Name
		if shortTag, ok := field.Tag.Lookup("short"); ok && short {
			fieldName = shortTag
		}

		fmt.Fprintf(w, "%s: ", fieldName)
		if value.Kind() == reflect.Struct {
			fmt.Fprint(w, "{ ")
			formatStruct(w, value.Interface(), short)
			fmt.Fprint(w, "} ")
		} else {
			fmt.Fprintf(w, "%v\n", value.Interface())
		}
	}
}
