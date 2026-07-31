package agentmodel

import (
	"fmt"
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

// ValidateComponentConfig validates a component's config (a decoded YAML/JSON object)
// against the schema's config field catalog. It returns nil when the schema does not
// describe this component's config (existence-only validation) and a list of problems
// otherwise. class is the component class ("receivers", ...) and name the component
// type ("otlp", ...).
func (s *RemoteConfigSchema) ValidateComponentConfig(
	class string, name string, config map[string]any,
) []ConfigValidationError {
	byName, ok := s.Spec.ComponentConfigs[class]
	if !ok {
		return nil
	}

	field, ok := byName[name]
	if !ok {
		return nil
	}

	return field.validate("", config)
}

// validate walks value against the field schema, collecting problems. It is lenient by
// design — it reports unknown object keys and coarse type mismatches, but treats a nil
// value and an "any"-typed field as always valid, since the goal is to catch typos and
// obvious shape errors, not to enforce every constraint.
func (f ConfigField) validate(path string, value any) []ConfigValidationError {
	if value == nil || f.Type == ConfigFieldTypeAny {
		return nil
	}

	switch f.Type {
	case ConfigFieldTypeObject:
		return f.validateObject(path, value)
	case ConfigFieldTypeMap:
		return f.validateMap(path, value)
	case ConfigFieldTypeArray:
		return f.validateArray(path, value)
	default:
		return f.validateScalar(path, value)
	}
}

func (f ConfigField) validateObject(path string, value any) []ConfigValidationError {
	asMap, ok := value.(map[string]any)
	if !ok {
		return []ConfigValidationError{{Path: path, Message: typeMismatch("object", value)}}
	}

	var problems []ConfigValidationError

	for key, sub := range asMap {
		child, known := f.Fields[key]
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

func (f ConfigField) validateMap(path string, value any) []ConfigValidationError {
	asMap, ok := value.(map[string]any)
	if !ok {
		return []ConfigValidationError{{Path: path, Message: typeMismatch("map", value)}}
	}

	if f.Elem == nil {
		return nil
	}

	var problems []ConfigValidationError

	for key, sub := range asMap {
		problems = append(problems, f.Elem.validate(join(path, key), sub)...)
	}

	return problems
}

func (f ConfigField) validateArray(path string, value any) []ConfigValidationError {
	asSlice, ok := value.([]any)
	if !ok {
		return []ConfigValidationError{{Path: path, Message: typeMismatch("array", value)}}
	}

	if f.Elem == nil {
		return nil
	}

	var problems []ConfigValidationError

	for i, sub := range asSlice {
		problems = append(problems, f.Elem.validate(fmt.Sprintf("%s[%d]", path, i), sub)...)
	}

	return problems
}

func (f ConfigField) validateScalar(path string, value any) []ConfigValidationError {
	if scalarMatches(f.Type, value) {
		return nil
	}

	return []ConfigValidationError{{Path: path, Message: typeMismatch(f.Type, value)}}
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
