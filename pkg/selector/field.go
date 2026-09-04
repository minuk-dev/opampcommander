package selector

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ErrUnsupportedField is the sentinel returned by [FieldSelector.Validate] when a
// selector names a field the resource does not support filtering on. Callers map
// it to 400 Bad Request; the message names the offending field and lists the
// supported ones, so a client never silently receives an unfiltered list.
var ErrUnsupportedField = errors.New("unsupported field selector field")

// FieldRequirement is a single constraint on one of a resource's own fields.
type FieldRequirement struct {
	// Field is the dotted path of the field, e.g. "status.connected".
	Field string
	// Operator is either [OpEquals] or [OpNotEquals]; field selectors have no
	// set-based or existence forms.
	Operator Operator
	// Value is the compared value, rendered as a string.
	Value string
}

// FieldSelector is a conjunction of field requirements. The zero value holds none
// and matches everything.
type FieldSelector []FieldRequirement

// ParseFields parses a comma-separated field selector expression such as
// "status.connected=true,metadata.namespace=prod". An empty expression yields an
// empty selector, which matches everything.
//
// Parsing does not check the fields against any allowlist — the resource being
// listed decides that, via [FieldSelector.Validate].
//
// Every failure wraps [ErrInvalidSelector].
func ParseFields(raw string) (FieldSelector, error) {
	entries, err := splitRequirements(raw)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, nil
	}

	selector := make(FieldSelector, 0, len(entries))

	for _, entry := range entries {
		requirement, err := parseFieldRequirement(entry)
		if err != nil {
			return nil, err
		}

		selector = append(selector, requirement)
	}

	return selector, nil
}

// Empty reports whether the selector constrains nothing.
func (s FieldSelector) Empty() bool {
	return len(s) == 0
}

// Fields returns the field paths the selector references, in order.
func (s FieldSelector) Fields() []string {
	fields := make([]string, 0, len(s))
	for _, requirement := range s {
		fields = append(fields, requirement.Field)
	}

	return fields
}

// Validate reports whether every referenced field is in allowed, returning an
// error wrapping [ErrUnsupportedField] — naming the first unsupported field and
// listing the allowed ones — otherwise.
func (s FieldSelector) Validate(allowed []string) error {
	for _, requirement := range s {
		if !slices.Contains(allowed, requirement.Field) {
			return fmt.Errorf("%w: %q; supported fields: %s",
				ErrUnsupportedField, requirement.Field, strings.Join(allowed, ", "))
		}
	}

	return nil
}

// Matches reports whether the given field projection satisfies every
// requirement.
//
// Unlike label selectors, a field is part of the resource's own shape rather
// than a user-supplied map: a field missing from the projection is treated as
// unset, so "field!=x" matches it.
func (s FieldSelector) Matches(fields map[string]string) bool {
	for _, requirement := range s {
		value, present := fields[requirement.Field]

		switch requirement.Operator {
		case OpEquals:
			if !present || value != requirement.Value {
				return false
			}
		case OpNotEquals:
			if present && value == requirement.Value {
				return false
			}
		case OpIn, OpNotIn, OpExists, OpNotExists:
			return false
		default:
			return false
		}
	}

	return true
}

// String renders the selector back into its parseable expression form.
func (s FieldSelector) String() string {
	parts := make([]string, 0, len(s))
	for _, requirement := range s {
		parts = append(parts, requirement.Field+string(requirement.Operator)+requirement.Value)
	}

	return strings.Join(parts, ",")
}

func parseFieldRequirement(entry string) (FieldRequirement, error) {
	//exhaustruct:ignore
	var zero FieldRequirement

	index, operator := findEqualityOperator(entry)
	if index < 0 {
		return zero, fmt.Errorf(
			"%w: %q is not a field requirement; expected field=value or field!=value", ErrInvalidSelector, entry)
	}

	field := strings.TrimSpace(entry[:index])
	value := strings.TrimSpace(entry[index+len(operator):])

	err := validateKey(field)
	if err != nil {
		return zero, err
	}

	err = validateValue(value)
	if err != nil {
		return zero, err
	}

	return FieldRequirement{
		Field:    field,
		Operator: canonicalEqualityOperator(operator),
		Value:    value,
	}, nil
}
