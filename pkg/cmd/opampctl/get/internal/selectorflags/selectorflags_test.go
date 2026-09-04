package selectorflags_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/internal/selectorflags"
)

// newFlags registers the shared flags on a throwaway command and parses argv,
// returning the populated flag set.
func newFlags(t *testing.T, argv ...string) *selectorflags.Flags {
	t.Helper()

	flags := &selectorflags.Flags{}

	//exhaustruct:ignore
	cmd := &cobra.Command{Use: "test"}
	flags.Register(cmd)

	require.NoError(t, cmd.ParseFlags(argv))

	return flags
}

func TestListOptions_NoFlagsAddsNothing(t *testing.T) {
	t.Parallel()

	opts, err := newFlags(t).ListOptions()
	require.NoError(t, err)
	assert.Empty(t, opts)
}

func TestListOptions_OneOptionPerSetFlag(t *testing.T) {
	t.Parallel()

	opts, err := newFlags(t,
		"-l", "env=prod",
		"--field-selector", "metadata.namespace=prod",
		"--name", "otel-",
	).ListOptions()
	require.NoError(t, err)
	assert.Len(t, opts, 3)
}

// A malformed expression has to fail here rather than reach the server, so the
// user sees which flag is wrong instead of a round-tripped 400.
func TestListOptions_RejectsMalformedExpressions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		argv []string
		want string
	}{
		"label":  {argv: []string{"-l", "env in prod"}, want: "invalid --selector"},
		"field":  {argv: []string{"--field-selector", "status.connected"}, want: "invalid --field-selector"},
		"setops": {argv: []string{"--field-selector", "spec.version in (1,2)"}, want: "invalid --field-selector"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := newFlags(t, test.argv...).ListOptions()
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.want)
		})
	}
}

func TestConstrainsField(t *testing.T) {
	t.Parallel()

	flags := newFlags(t, "--field-selector", "status.connected=false,metadata.namespace=prod")

	assert.True(t, flags.ConstrainsField("status.connected"))
	assert.True(t, flags.ConstrainsField("metadata.namespace"))
	assert.False(t, flags.ConstrainsField("status.healthy"))
	assert.False(t, newFlags(t).ConstrainsField("status.connected"))
}

func TestLocalFilter_MatchesNameAndLabels(t *testing.T) {
	t.Parallel()

	filter, err := newFlags(t, "-l", "env=prod,!deprecated", "--name", "otel-").LocalFilter()
	require.NoError(t, err)

	assert.True(t, filter.Matches("otel-1", map[string]string{"env": "prod"}))
	assert.False(t, filter.Matches("otel-1", map[string]string{"env": "dev"}))
	assert.False(t, filter.Matches("other-1", map[string]string{"env": "prod"}))
	assert.False(t, filter.Matches("otel-1", map[string]string{"env": "prod", "deprecated": "true"}))
}

func TestLocalFilter_EmptyMatchesEverything(t *testing.T) {
	t.Parallel()

	filter, err := newFlags(t).LocalFilter()
	require.NoError(t, err)

	assert.True(t, filter.Matches("anything", nil))
}

// A listing that filters in the process has no field projection to evaluate, so
// --field-selector must fail rather than be dropped: silently returning the whole
// list is the failure mode the server-side selectors exist to prevent.
func TestLocalFilter_RejectsFieldSelector(t *testing.T) {
	t.Parallel()

	_, err := newFlags(t, "--field-selector", "metadata.namespace=prod").LocalFilter()
	require.ErrorIs(t, err, selectorflags.ErrUnsupportedLocally)
}
