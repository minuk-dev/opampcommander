package v1

import (
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	//nolint:exhaustruct
	_ json.Marshaler   = Time{}
	_ json.Unmarshaler = (*Time)(nil)
	//nolint:exhaustruct
	_ yaml.Marshaler   = Time{}
	_ yaml.Unmarshaler = (*Time)(nil)
)

// Time is a wrapper around time.Time which supports correct
// marshaling to YAML and JSON.  Wrappers are provided for many
// of the factory methods that the time package offers.
//
//nolint:recvcheck
type Time struct {
	time.Time
}

// NewTime returns a wrapped instance of the provided time.
func NewTime(time time.Time) Time {
	return Time{time}
}

// DeepCopyInto creates a deep-copy of the Time value.  The underlying time.Time
// type is effectively immutable in the time API, so it is safe to
// copy-by-assign, despite the presence of (unexported) Pointer fields.
func (t *Time) DeepCopyInto(out *Time) {
	*out = *t
}

// Date returns the Time corresponding to the supplied parameters
// by wrapping time.Date.
//
//nolint:revive,predeclared
func Date(year int, month time.Month, day, hour, min, sec, nsec int, loc *time.Location) Time {
	return Time{time.Date(year, month, day, hour, min, sec, nsec, loc)}
}

// Now returns the current local time.
func Now() Time {
	return Time{time.Now()}
}

// IsZero returns true if the value is nil or time is zero.
func (t *Time) IsZero() bool {
	if t == nil {
		return true
	}

	return t.Time.IsZero()
}

// Before reports whether the time instant t is before u.
func (t *Time) Before(u *Time) bool {
	if t != nil && u != nil {
		return t.Time.Before(u.Time)
	}

	return false
}

// Equal reports whether the time instant t is equal to u.
//
//nolint:varnamelen
func (t *Time) Equal(u *Time) bool {
	if t == nil && u == nil {
		return true
	}

	if t != nil && u != nil {
		return t.Time.Equal(u.Time)
	}

	return false
}

// Unix returns the local time corresponding to the given Unix time
// by wrapping time.Unix.
func Unix(sec int64, nsec int64) Time {
	return Time{time.Unix(sec, nsec)}
}

// Rfc3339Copy returns a copy of the Time at second-level precision.
func (t Time) Rfc3339Copy() Time {
	copied, _ := time.Parse(time.RFC3339, t.Format(time.RFC3339))

	return Time{copied}
}

// UnmarshalJSON implements the json.Unmarshaller interface.
//
//nolint:varnamelen,gosmopolitan
func (t *Time) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		t.Time = time.Time{}

		return nil
	}

	var str string

	err := json.Unmarshal(b, &str)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json time: %w", err)
	}

	pt, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return fmt.Errorf("failed to parse time %q: %w", str, err)
	}

	t.Time = pt.Local()

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
//
//nolint:mnd
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		// Encode unset/nil objects as JSON's "null".
		return []byte("null"), nil
	}

	buf := make([]byte, 0, len(time.RFC3339)+2)
	buf = append(buf, '"')
	// time cannot contain non escapable JSON characters
	buf = t.UTC().AppendFormat(buf, time.RFC3339)
	buf = append(buf, '"')

	return buf, nil
}

// UnmarshalYAML implements [yaml.Unmarshaler].
//
//nolint:gosmopolitan
func (t *Time) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode && value.Tag == "!!null" {
		t.Time = time.Time{}

		return nil
	}

	var str string

	err := value.Decode(&str)
	if err != nil {
		return fmt.Errorf("failed to decode yaml time: %w", err)
	}

	pt, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return fmt.Errorf("failed to parse time %q: %w", str, err)
	}

	t.Time = pt.Local()

	return nil
}

// MarshalYAML implements [yaml.Marshaler].
//
//nolint:mnd
func (t Time) MarshalYAML() (interface{}, error) {
	if t.IsZero() {
		// Encode unset/nil objects as JSON's "null".
		return []byte("null"), nil
	}

	buf := make([]byte, 0, len(time.RFC3339)+2)
	buf = append(buf, '"')
	// time cannot contain non escapable JSON characters
	buf = t.UTC().AppendFormat(buf, time.RFC3339)
	buf = append(buf, '"')

	return buf, nil
}
