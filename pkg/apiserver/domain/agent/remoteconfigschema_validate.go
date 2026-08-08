package agentmodel

import (
	"fmt"
	"slices"
	"strings"
)

// ConfigValidationError is a single problem found while validating a component config
// against its schema, located at a dotted path within the component config.
type ConfigValidationError struct {
	// Path is the dotted field path (empty at the component root).
	Path string
	// Message describes the problem (unknown field, type mismatch, ...).
	Message string
}

// Error implements the error interface.
func (e ConfigValidationError) Error() string {
	if e.Path == "" {
		return e.Message
	}

	return e.Path + ": " + e.Message
}

// ValidateComponent validates one component of a collector config against the schema:
// that the collector build ships it, and — when the schema describes its settings —
// that config matches them field by field. class is the component class
// ("receivers", ...) and name the component type ("otlp", ...).
//
// It is lenient wherever the schema is silent: a class the catalog does not describe,
// a component with no field schema, and an open-ended field are all accepted, so a
// shallow catalog reports nothing rather than reporting everything.
func (s *RemoteConfigSchema) ValidateComponent(
	class string, name string, config map[string]any,
) []ConfigValidationError {
	byName, ok := s.Spec.Components[class]
	if !ok {
		return nil
	}

	component, ok := byName[name]
	if !ok {
		return []ConfigValidationError{{
			Path:    "",
			Message: fmt.Sprintf("unknown %s %q for %s %s", singularClass(class), name, s.Spec.Binary, s.Spec.Version),
		}}
	}

	if component.Fields == nil {
		return nil
	}

	return component.Fields.validate("", config)
}

// ValidateSignalSupport reports a problem when the component is used in a pipeline of
// the given signal but does not handle that signal. Unknown components and components
// whose signals the catalog does not record are accepted.
func (s *RemoteConfigSchema) ValidateSignalSupport(
	class string, name string, signal string,
) []ConfigValidationError {
	component, ok := s.Component(class, name)
	if !ok || component.SupportsSignal(signal) {
		return nil
	}

	return []ConfigValidationError{{
		Path: "",
		Message: fmt.Sprintf("%s %q does not support %s (supports %s)",
			singularClass(class), name, signal, strings.Join(component.Signals, ", ")),
	}}
}

// ValidateConnectorPair reports a problem when the connector is used to convert
// fromSignal into toSignal but does not support that conversion. Unknown connectors
// and connectors whose pairs the catalog does not record are accepted.
func (s *RemoteConfigSchema) ValidateConnectorPair(
	class string, name string, fromSignal string, toSignal string,
) []ConfigValidationError {
	component, ok := s.Component(class, name)
	if !ok || component.SupportsPair(fromSignal, toSignal) {
		return nil
	}

	return []ConfigValidationError{{
		Path: "",
		Message: fmt.Sprintf("%s %q does not convert %s to %s",
			singularClass(class), name, fromSignal, toSignal),
	}}
}

// singularClass renders a class key ("receivers") for use in a message about a single
// component ("receiver"). Class keys are open-ended, so anything that does not end in
// "s" is left as is.
func singularClass(class string) string {
	return strings.TrimSuffix(class, "s")
}

// validate walks value against the field schema, collecting problems. It is lenient by
// design — it reports unknown keys of a closed map, values outside an enum, and coarse
// type mismatches, but treats a nil value and an untyped field as always valid, since
// the goal is to catch typos and obvious shape errors, not to enforce every constraint.
func (f ConfigField) validate(path string, value any) []ConfigValidationError {
	if value == nil || f.Type == "" {
		return nil
	}

	switch f.Type {
	case ConfigFieldTypeMap:
		return f.validateMap(path, value)
	case ConfigFieldTypeList:
		return f.validateList(path, value)
	default:
		return f.validateScalar(path, value)
	}
}

func (f ConfigField) validateMap(path string, value any) []ConfigValidationError {
	asMap, ok := value.(map[string]any)
	if !ok {
		return []ConfigValidationError{{Path: path, Message: typeMismatch("map", value)}}
	}

	// An open field, or one whose settings the schema does not spell out, accepts any
	// key: there is nothing to check it against.
	if f.Open || len(f.Children) == 0 {
		return nil
	}

	var problems []ConfigValidationError

	for key, sub := range asMap {
		child, known := f.Children[key]
		if !known {
			problems = append(problems, ConfigValidationError{
				Path:    join(path, key),
				Message: fmt.Sprintf("unknown field %q", key),
			})

			continue
		}

		problems = append(problems, child.validate(join(path, key), sub)...)
	}

	return problems
}

func (f ConfigField) validateList(path string, value any) []ConfigValidationError {
	asSlice, ok := value.([]any)
	if !ok {
		return []ConfigValidationError{{Path: path, Message: typeMismatch("list", value)}}
	}

	item, ok := f.Children[ConfigFieldItemKey]
	if !ok {
		return nil
	}

	var problems []ConfigValidationError

	for i, sub := range asSlice {
		problems = append(problems, item.validate(fmt.Sprintf("%s[%d]", path, i), sub)...)
	}

	return problems
}

func (f ConfigField) validateScalar(path string, value any) []ConfigValidationError {
	if !scalarMatches(f.Type, value) {
		return []ConfigValidationError{{Path: path, Message: typeMismatch(f.Type, value)}}
	}

	return f.validateEnum(path, value)
}

// validateEnum reports a value outside the field's allowed set. Only strings are
// checked: the enums the collector publishes are string-valued.
func (f ConfigField) validateEnum(path string, value any) []ConfigValidationError {
	asString, isString := value.(string)
	if len(f.Enum) == 0 || !isString || slices.Contains(f.Enum, asString) {
		return nil
	}

	return []ConfigValidationError{{
		Path:    path,
		Message: fmt.Sprintf("invalid value %q (want one of: %s)", asString, strings.Join(f.Enum, ", ")),
	}}
}

// scalarMatches reports whether value is acceptable for a scalar field type. It is
// lenient: integers satisfy floats, either numeric form satisfies a duration (in
// addition to the usual string form), and an object/array never satisfies a scalar.
func scalarMatches(fieldType string, value any) bool {
	switch fieldType {
	case ConfigFieldTypeString:
		_, ok := value.(string)

		return ok
	case ConfigFieldTypeBool:
		_, ok := value.(bool)

		return ok
	case ConfigFieldTypeInt:
		return isInteger(value)
	case ConfigFieldTypeFloat:
		return isNumber(value)
	case ConfigFieldTypeDuration:
		_, isString := value.(string)

		return isString || isNumber(value)
	default:
		// Unknown field type: don't flag it.
		return true
	}
}

func isInteger(value any) bool {
	switch typed := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return typed == float64(int64(typed))
	case float32:
		return typed == float32(int32(typed))
	default:
		return false
	}
}

func isNumber(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	default:
		return false
	}
}

func typeMismatch(want string, value any) string {
	return fmt.Sprintf("expected %s, got %T", want, value)
}

func join(path, key string) string {
	if path == "" {
		return key
	}

	return path + "." + key
}
