package selector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/selector"
)

func TestParseFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		raw      string
		expected selector.FieldSelector
	}{
		{
			name:     "empty",
			raw:      "",
			expected: nil,
		},
		{
			name: "equality",
			raw:  "status.connected=true",
			expected: selector.FieldSelector{
				{Field: "status.connected", Operator: selector.OpEquals, Value: "true"},
			},
		},
		{
			name: "inequality",
			raw:  "status.connected!=true",
			expected: selector.FieldSelector{
				{Field: "status.connected", Operator: selector.OpNotEquals, Value: "true"},
			},
		},
		{
			name: "conjunction",
			raw:  "status.connected=true,metadata.namespace=prod",
			expected: selector.FieldSelector{
				{Field: "status.connected", Operator: selector.OpEquals, Value: "true"},
				{Field: "metadata.namespace", Operator: selector.OpEquals, Value: "prod"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := selector.ParseFields(testCase.raw)
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, parsed)
		})
	}
}

func TestParseFields_Invalid(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"status.connected",           // no operator: field selectors have no existence form
		"status.connected in (true)", // no set-based form either
		"=true",
		"status.connected=true,",
	} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			_, err := selector.ParseFields(raw)
			require.ErrorIs(t, err, selector.ErrInvalidSelector)
		})
	}
}

func TestFieldSelector_Validate(t *testing.T) {
	t.Parallel()

	allowed := []string{"status.connected", "metadata.namespace"}

	parsed, err := selector.ParseFields("status.connected=true")
	require.NoError(t, err)
	require.NoError(t, parsed.Validate(allowed))

	parsed, err = selector.ParseFields("spec.secret=hunter2")
	require.NoError(t, err)

	err = parsed.Validate(allowed)
	require.ErrorIs(t, err, selector.ErrUnsupportedField)
	assert.Contains(t, err.Error(), "spec.secret", "the error must name the offending field")
	assert.Contains(t, err.Error(), "status.connected", "the error must list the supported fields")
}

func TestFieldSelector_ValidateRejectsEveryFieldWhenNoneAllowed(t *testing.T) {
	t.Parallel()

	parsed, err := selector.ParseFields("metadata.name=a")
	require.NoError(t, err)
	require.ErrorIs(t, parsed.Validate(nil), selector.ErrUnsupportedField)
}

func TestFieldSelector_Matches(t *testing.T) {
	t.Parallel()

	fields := map[string]string{"status.connected": "true", "metadata.namespace": "prod"}

	testCases := []struct {
		name     string
		raw      string
		expected bool
	}{
		{name: "empty matches everything", raw: "", expected: true},
		{name: "equality hit", raw: "status.connected=true", expected: true},
		{name: "equality miss", raw: "status.connected=false", expected: false},
		{name: "inequality hit", raw: "status.connected!=false", expected: true},
		{name: "inequality miss", raw: "status.connected!=true", expected: false},
		{name: "unset field is not equal to anything", raw: "spec.platform=vm", expected: false},
		{name: "unset field satisfies inequality", raw: "spec.platform!=vm", expected: true},
		{name: "conjunction", raw: "status.connected=true,metadata.namespace=prod", expected: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := selector.ParseFields(testCase.raw)
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, parsed.Matches(fields))
		})
	}
}

func TestFieldSelector_FieldsAndString(t *testing.T) {
	t.Parallel()

	parsed, err := selector.ParseFields("status.connected=true,metadata.namespace!=prod")
	require.NoError(t, err)

	assert.Equal(t, []string{"status.connected", "metadata.namespace"}, parsed.Fields())
	assert.Equal(t, "status.connected=true,metadata.namespace!=prod", parsed.String())
	assert.False(t, parsed.Empty())
}
