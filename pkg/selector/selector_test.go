package selector_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/selector"
)

func TestParseLabels(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		raw      string
		expected selector.LabelSelector
	}{
		{
			name:     "empty",
			raw:      "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			raw:      "   ",
			expected: nil,
		},
		{
			name: "equality",
			raw:  "env=prod",
			expected: selector.LabelSelector{
				{Key: "env", Operator: selector.OpEquals, Values: []string{"prod"}},
			},
		},
		{
			name: "double equals is equality",
			raw:  "env==prod",
			expected: selector.LabelSelector{
				{Key: "env", Operator: selector.OpEquals, Values: []string{"prod"}},
			},
		},
		{
			name: "inequality",
			raw:  "tier!=canary",
			expected: selector.LabelSelector{
				{Key: "tier", Operator: selector.OpNotEquals, Values: []string{"canary"}},
			},
		},
		{
			name: "empty value is allowed",
			raw:  "env=",
			expected: selector.LabelSelector{
				{Key: "env", Operator: selector.OpEquals, Values: []string{""}},
			},
		},
		{
			name: "in",
			raw:  "tier in (canary, prod)",
			expected: selector.LabelSelector{
				{Key: "tier", Operator: selector.OpIn, Values: []string{"canary", "prod"}},
			},
		},
		{
			name: "notin without space before paren",
			raw:  "tier notin(canary)",
			expected: selector.LabelSelector{
				{Key: "tier", Operator: selector.OpNotIn, Values: []string{"canary"}},
			},
		},
		{
			name: "exists",
			raw:  "deprecated",
			expected: selector.LabelSelector{
				{Key: "deprecated", Operator: selector.OpExists, Values: nil},
			},
		},
		{
			name: "not exists",
			raw:  "!deprecated",
			expected: selector.LabelSelector{
				{Key: "deprecated", Operator: selector.OpNotExists, Values: nil},
			},
		},
		{
			name: "conjunction keeps commas inside a value set together",
			raw:  "env=prod,tier notin (canary,dev),!deprecated",
			expected: selector.LabelSelector{
				{Key: "env", Operator: selector.OpEquals, Values: []string{"prod"}},
				{Key: "tier", Operator: selector.OpNotIn, Values: []string{"canary", "dev"}},
				{Key: "deprecated", Operator: selector.OpNotExists, Values: nil},
			},
		},
		{
			name: "surrounding whitespace is trimmed",
			raw:  "  env = prod ,  tier != canary  ",
			expected: selector.LabelSelector{
				{Key: "env", Operator: selector.OpEquals, Values: []string{"prod"}},
				{Key: "tier", Operator: selector.OpNotEquals, Values: []string{"canary"}},
			},
		},
		{
			name: "dotted opentelemetry attribute key",
			raw:  "service.namespace=payments",
			expected: selector.LabelSelector{
				{Key: "service.namespace", Operator: selector.OpEquals, Values: []string{"payments"}},
			},
		},
		{
			name: "prefixed key",
			raw:  "app.kubernetes.io/name=otelcol",
			expected: selector.LabelSelector{
				{Key: "app.kubernetes.io/name", Operator: selector.OpEquals, Values: []string{"otelcol"}},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := selector.ParseLabels(testCase.raw)
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, parsed)
		})
	}
}

func TestParseLabels_Invalid(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		raw  string
	}{
		{name: "empty requirement", raw: "env=prod,,tier=web"},
		{name: "trailing comma", raw: "env=prod,"},
		{name: "empty key", raw: "=prod"},
		{name: "empty key on not-exists", raw: "!"},
		{name: "unbalanced open paren", raw: "tier in (canary"},
		{name: "unbalanced close paren", raw: "tier in canary)"},
		{name: "empty value set", raw: "tier in ()"},
		{name: "key with a mongo operator prefix", raw: "$where=1"},
		{name: "key with a space", raw: "my key=value"},
		{name: "too many requirements", raw: strings.TrimSuffix(strings.Repeat("a=b,", 40), ",")},
		{name: "value with a control character", raw: "env=pr\x00od"},
		{name: "over-long key", raw: strings.Repeat("a", 400) + "=b"},
		{name: "over-long value", raw: "env=" + strings.Repeat("a", 300)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := selector.ParseLabels(testCase.raw)
			require.ErrorIs(t, err, selector.ErrInvalidSelector)
		})
	}
}

func TestLabelSelector_Matches(t *testing.T) {
	t.Parallel()

	labels := map[string]string{"env": "prod", "tier": "web"}

	testCases := []struct {
		name     string
		raw      string
		expected bool
	}{
		{name: "empty selector matches everything", raw: "", expected: true},
		{name: "equality hit", raw: "env=prod", expected: true},
		{name: "equality miss", raw: "env=dev", expected: false},
		{name: "inequality on a different value", raw: "env!=dev", expected: true},
		{name: "inequality on the same value", raw: "env!=prod", expected: false},
		{name: "inequality on an absent key matches", raw: "region!=eu", expected: true},
		{name: "in hit", raw: "tier in (web,api)", expected: true},
		{name: "in miss", raw: "tier in (api)", expected: false},
		{name: "in on an absent key misses", raw: "region in (eu)", expected: false},
		{name: "notin on an absent key matches", raw: "region notin (eu)", expected: true},
		{name: "notin on a listed value misses", raw: "tier notin (web)", expected: false},
		{name: "exists", raw: "env", expected: true},
		{name: "exists on an absent key", raw: "region", expected: false},
		{name: "not exists", raw: "!region", expected: true},
		{name: "not exists on a present key", raw: "!env", expected: false},
		{name: "conjunction all satisfied", raw: "env=prod,tier in (web),!region", expected: true},
		{name: "conjunction one unsatisfied", raw: "env=prod,tier in (api)", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := selector.ParseLabels(testCase.raw)
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, parsed.Matches(labels))
		})
	}
}

func TestLabelSelector_MatchesNilLabels(t *testing.T) {
	t.Parallel()

	parsed, err := selector.ParseLabels("env!=prod,!tier")
	require.NoError(t, err)
	assert.True(t, parsed.Matches(nil), "negative requirements must hold for a resource with no labels")

	parsed, err = selector.ParseLabels("env=prod")
	require.NoError(t, err)
	assert.False(t, parsed.Matches(nil))
}

func TestLabelSelector_StringRoundTrips(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"env=prod",
		"tier!=canary",
		"tier in (canary,dev)",
		"tier notin (canary,dev)",
		"deprecated",
		"!deprecated",
		"env=prod,tier notin (canary,dev),!deprecated",
	} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			parsed, err := selector.ParseLabels(raw)
			require.NoError(t, err)
			assert.Equal(t, raw, parsed.String())

			reparsed, err := selector.ParseLabels(parsed.String())
			require.NoError(t, err)
			assert.Equal(t, parsed, reparsed)
		})
	}
}

func TestLabelSelector_Empty(t *testing.T) {
	t.Parallel()

	parsed, err := selector.ParseLabels("")
	require.NoError(t, err)
	assert.True(t, parsed.Empty())

	parsed, err = selector.ParseLabels("env=prod")
	require.NoError(t, err)
	assert.False(t, parsed.Empty())
}

// An aggregate may carry its labels in more than one map — an agent's
// description is split into identifying and non-identifying attributes — and one
// selector has to reach all of them.
func TestMatchesAny_UnionAcrossLabelMaps(t *testing.T) {
	t.Parallel()

	identifying := map[string]string{"service.name": "otel-collector"}
	nonIdentifying := map[string]string{"os.type": "linux"}

	tests := map[string]struct {
		expression string
		want       bool
	}{
		"first map satisfies it":                 {"service.name=otel-collector", true},
		"second map satisfies it":                {"os.type=linux", true},
		"a conjunction spanning both":            {"service.name=otel-collector,os.type=linux", true},
		"neither map satisfies it":               {"os.type=windows", false},
		"negation holds when neither says so":    {"os.type!=windows", true},
		"negation fails when one map says so":    {"os.type!=linux", false},
		"negation holds for a key in neither":    {"cloud.provider!=aws", true},
		"existence holds for a key in either":    {"os.type", true},
		"non-existence fails for a key in one":   {"!os.type", false},
		"non-existence holds for a key in none":  {"!cloud.provider", true},
		"in holds when either map has the value": {"os.type in (linux,darwin)", true},
		"notin fails when one map has it":        {"os.type notin (linux)", false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			parsed, err := selector.ParseLabels(test.expression)
			require.NoError(t, err)
			assert.Equal(t, test.want, parsed.MatchesAny(identifying, nonIdentifying))
		})
	}
}

// With a single map MatchesAny must be indistinguishable from Matches, or the
// aggregates that carry one map would quietly change behaviour.
func TestMatchesAny_SingleMapAgreesWithMatches(t *testing.T) {
	t.Parallel()

	labels := map[string]string{"env": "prod", "tier": "web"}

	for _, expression := range []string{
		"env=prod", "env!=prod", "env in (prod,stg)", "env notin (prod)",
		"tier", "!tier", "missing!=x", "!missing", "env=prod,tier!=canary",
	} {
		parsed, err := selector.ParseLabels(expression)
		require.NoError(t, err)
		assert.Equal(t, parsed.Matches(labels), parsed.MatchesAny(labels), expression)
	}
}

// Every negative operator is exactly the negation of its positive twin. Both
// adapters rely on that to translate three forms instead of six.
func TestPositive(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		expression   string
		wantOperator selector.Operator
		wantNegated  bool
	}{
		"equals":     {"env=prod", selector.OpEquals, false},
		"not equals": {"env!=prod", selector.OpEquals, true},
		"in":         {"env in (prod)", selector.OpIn, false},
		"notin":      {"env notin (prod)", selector.OpIn, true},
		"exists":     {"env", selector.OpExists, false},
		"not exists": {"!env", selector.OpExists, true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			parsed, err := selector.ParseLabels(test.expression)
			require.NoError(t, err)
			require.Len(t, parsed, 1)

			positive, negated := parsed[0].Positive()
			assert.Equal(t, test.wantOperator, positive.Operator)
			assert.Equal(t, test.wantNegated, negated)
			assert.Equal(t, parsed[0].Key, positive.Key)
			assert.Equal(t, parsed[0].Values, positive.Values)
		})
	}
}
