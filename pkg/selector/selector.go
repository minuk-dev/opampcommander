package selector

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Operator is the comparison a requirement applies to a label or field value.
type Operator string

// Supported operators. Equals and NotEquals are shared with field selectors; the
// rest are label-selector only.
const (
	// OpEquals matches when the key is present and its value is the given one.
	// Written "key=value" or "key==value".
	OpEquals Operator = "="
	// OpNotEquals matches when the key is absent, or present with a different
	// value. Written "key!=value".
	OpNotEquals Operator = "!="
	// OpIn matches when the key is present and its value is one of the given ones.
	// Written "key in (a,b)".
	OpIn Operator = "in"
	// OpNotIn matches when the key is absent, or present with a value that is none
	// of the given ones. Written "key notin (a,b)".
	OpNotIn Operator = "notin"
	// OpExists matches when the key is present, whatever its value. Written "key".
	OpExists Operator = "exists"
	// OpNotExists matches when the key is absent. Written "!key".
	OpNotExists Operator = "!"
)

// ErrInvalidSelector is the sentinel every parse failure wraps, so callers can
// map a malformed selector to 400 Bad Request without inspecting the message.
var ErrInvalidSelector = errors.New("invalid selector")

// Limits on what a single selector may express. They bound the query a caller can
// make the datastore build; the values are far above any realistic selector.
const (
	// MaxRequirements is the most requirements one selector may hold.
	MaxRequirements = 32
	// MaxValues is the most values one "in"/"notin" requirement may list.
	MaxValues = 64
	// maxKeyLength is the longest accepted key: a 253-character DNS-subdomain
	// prefix, a slash, and a 63-character name.
	maxKeyLength = 317
	// maxValueLength is the longest accepted value.
	maxValueLength = 253
)

// keyPattern accepts an optional DNS-subdomain prefix followed by a name, e.g.
// "env", "service.namespace" or "app.kubernetes.io/name". It deliberately admits
// dots inside the name so OpenTelemetry attribute keys are expressible.
var keyPattern = regexp.MustCompile(
	`^([a-z0-9]([-a-z0-9.]*[a-z0-9])?/)?[A-Za-z0-9]([-A-Za-z0-9_.]*[A-Za-z0-9])?$`)

// setPattern matches the set-based forms "key in (a,b)" and "key notin (a,b)".
var setPattern = regexp.MustCompile(`^([^\s=!(),]+)\s+(in|notin)\s*\(([^()]*)\)$`)

// Requirement is a single label constraint.
type Requirement struct {
	// Key is the label key the requirement constrains.
	Key string
	// Operator is how Values (if any) are compared against the label's value.
	Operator Operator
	// Values holds the compared values: exactly one for OpEquals/OpNotEquals, one
	// or more for OpIn/OpNotIn, and none for OpExists/OpNotExists.
	Values []string
}

// LabelSelector is a conjunction of requirements: a resource matches only if it
// satisfies every one. The zero value holds no requirements and matches
// everything.
type LabelSelector []Requirement

// ParseLabels parses a comma-separated label selector expression. An empty (or
// all-whitespace) expression yields an empty selector, which matches everything.
//
// Every failure wraps [ErrInvalidSelector].
func ParseLabels(raw string) (LabelSelector, error) {
	entries, err := splitRequirements(raw)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, nil
	}

	selector := make(LabelSelector, 0, len(entries))

	for _, entry := range entries {
		requirement, err := parseRequirement(entry)
		if err != nil {
			return nil, err
		}

		selector = append(selector, requirement)
	}

	return selector, nil
}

// Empty reports whether the selector constrains nothing, i.e. matches every
// resource.
func (s LabelSelector) Empty() bool {
	return len(s) == 0
}

// Matches reports whether the given label map satisfies every requirement.
func (s LabelSelector) Matches(labels map[string]string) bool {
	for _, requirement := range s {
		if !requirement.Matches(labels) {
			return false
		}
	}

	return true
}

// String renders the selector back into its parseable expression form.
func (s LabelSelector) String() string {
	parts := make([]string, 0, len(s))
	for _, requirement := range s {
		parts = append(parts, requirement.String())
	}

	return strings.Join(parts, ",")
}

// Matches reports whether the given label map satisfies the requirement.
//
// The negative operators (OpNotEquals, OpNotIn) match an absent key, mirroring
// Kubernetes: "tier!=canary" selects resources with no tier label at all.
func (r Requirement) Matches(labels map[string]string) bool {
	value, present := labels[r.Key]

	switch r.Operator {
	case OpEquals:
		return present && len(r.Values) > 0 && value == r.Values[0]
	case OpNotEquals:
		return !present || len(r.Values) == 0 || value != r.Values[0]
	case OpIn:
		return present && slices.Contains(r.Values, value)
	case OpNotIn:
		return !present || !slices.Contains(r.Values, value)
	case OpExists:
		return present
	case OpNotExists:
		return !present
	default:
		return false
	}
}

// String renders the requirement back into its parseable expression form.
func (r Requirement) String() string {
	switch r.Operator {
	case OpExists:
		return r.Key
	case OpNotExists:
		return "!" + r.Key
	case OpIn, OpNotIn:
		return fmt.Sprintf("%s %s (%s)", r.Key, r.Operator, strings.Join(r.Values, ","))
	case OpEquals, OpNotEquals:
		return r.Key + string(r.Operator) + firstValue(r.Values)
	default:
		return ""
	}
}

func parseRequirement(entry string) (Requirement, error) {
	//exhaustruct:ignore
	var zero Requirement

	if strings.HasPrefix(entry, "!") {
		key := strings.TrimSpace(entry[1:])

		err := validateKey(key)
		if err != nil {
			return zero, err
		}

		return Requirement{Key: key, Operator: OpNotExists, Values: nil}, nil
	}

	if match := setPattern.FindStringSubmatch(entry); match != nil {
		return parseSetRequirement(match[1], Operator(match[2]), match[3])
	}

	if index, operator := findEqualityOperator(entry); index >= 0 {
		key := strings.TrimSpace(entry[:index])
		value := strings.TrimSpace(entry[index+len(operator):])

		err := validateKey(key)
		if err != nil {
			return zero, err
		}

		err = validateValue(value)
		if err != nil {
			return zero, err
		}

		return Requirement{Key: key, Operator: canonicalEqualityOperator(operator), Values: []string{value}}, nil
	}

	err := validateKey(entry)
	if err != nil {
		return zero, err
	}

	return Requirement{Key: entry, Operator: OpExists, Values: nil}, nil
}

func parseSetRequirement(key string, operator Operator, rawValues string) (Requirement, error) {
	//exhaustruct:ignore
	var zero Requirement

	err := validateKey(key)
	if err != nil {
		return zero, err
	}

	if strings.TrimSpace(rawValues) == "" {
		return zero, fmt.Errorf("%w: %q needs at least one value", ErrInvalidSelector, operator)
	}

	values := make([]string, 0, strings.Count(rawValues, ",")+1)

	for rawValue := range strings.SplitSeq(rawValues, ",") {
		value := strings.TrimSpace(rawValue)

		err = validateValue(value)
		if err != nil {
			return zero, err
		}

		values = append(values, value)
	}

	if len(values) > MaxValues {
		return zero, fmt.Errorf("%w: %q lists %d values, at most %d are allowed",
			ErrInvalidSelector, key, len(values), MaxValues)
	}

	return Requirement{Key: key, Operator: operator, Values: values}, nil
}

// findEqualityOperator returns the index and text of the first "=", "==" or "!="
// in entry, or -1 when it holds none. A leading "!" has already been consumed as
// OpNotExists by the caller, so any "!" reaching here must open a "!=".
func findEqualityOperator(entry string) (int, string) {
	for index := range len(entry) {
		switch entry[index] {
		case '=':
			if index+1 < len(entry) && entry[index+1] == '=' {
				return index, "=="
			}

			return index, "="
		case '!':
			if index+1 < len(entry) && entry[index+1] == '=' {
				return index, "!="
			}
		}
	}

	return -1, ""
}

func canonicalEqualityOperator(operator string) Operator {
	if operator == "!=" {
		return OpNotEquals
	}

	return OpEquals
}

// splitRequirements splits a selector expression on the commas that separate
// requirements, leaving the commas inside an "in (…)" value list alone. It
// returns the trimmed, non-empty entries.
func splitRequirements(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	entries := make([]string, 0, strings.Count(raw, ",")+1)
	depth := 0
	start := 0

	for index := range len(raw) {
		switch raw[index] {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("%w: unbalanced %q in %q", ErrInvalidSelector, ")", raw)
			}
		case ',':
			if depth == 0 {
				entries = append(entries, raw[start:index])
				start = index + 1
			}
		}
	}

	if depth != 0 {
		return nil, fmt.Errorf("%w: unbalanced %q in %q", ErrInvalidSelector, "(", raw)
	}

	entries = append(entries, raw[start:])

	trimmed := make([]string, 0, len(entries))

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			return nil, fmt.Errorf("%w: empty requirement in %q", ErrInvalidSelector, raw)
		}

		trimmed = append(trimmed, entry)
	}

	if len(trimmed) > MaxRequirements {
		return nil, fmt.Errorf("%w: %d requirements given, at most %d are allowed",
			ErrInvalidSelector, len(trimmed), MaxRequirements)
	}

	return trimmed, nil
}

func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: empty key", ErrInvalidSelector)
	}

	if len(key) > maxKeyLength {
		return fmt.Errorf("%w: key %q is longer than %d characters", ErrInvalidSelector, key, maxKeyLength)
	}

	if !keyPattern.MatchString(key) {
		return fmt.Errorf("%w: %q is not a valid key", ErrInvalidSelector, key)
	}

	return nil
}

// validateValue accepts any printable value short enough to be a label value.
// Values reach the datastore in value position — never as a field path or an
// operator — so they need no structural restriction, only a bound.
func validateValue(value string) error {
	if len(value) > maxValueLength {
		return fmt.Errorf("%w: value %q is longer than %d characters", ErrInvalidSelector, value, maxValueLength)
	}

	for _, r := range value {
		if r < ' ' || r == 0x7F {
			return fmt.Errorf("%w: value %q contains a control character", ErrInvalidSelector, value)
		}
	}

	return nil
}

func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}

	return values[0]
}
