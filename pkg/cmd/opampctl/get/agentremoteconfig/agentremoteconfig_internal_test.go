package agentremoteconfig

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
)

func TestToFormattedAgentRemoteConfig_SchemaRefs(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		refs   []string
		source string
		want   string
	}{
		{name: "no references", want: "none"},
		{name: "legacy references", refs: []string{"schema-a", "schema-b"}, want: "unknown"},
		{name: "automatically resolved", refs: []string{"schema-a", "schema-b"},
			source: "auto", want: "auto"},
		{name: "explicitly set", refs: []string{"schema-a", "schema-b"}, source: "explicit", want: "explicit"},
		{name: "future source", refs: []string{"schema-a"}, source: "future", want: "future"},
		{name: "cleared references", source: "explicit", want: "none"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var resource v1.AgentRemoteConfig

			resource.Metadata.Name = "cfg"

			resource.Spec.SchemaRefs = tt.refs
			resource.Status.SchemaRefsSource = tt.source

			var options CommandOptions

			formatted := options.toFormattedAgentRemoteConfig(resource)
			assert.Equal(t, tt.refs, formatted.SchemaRefs)
			assert.Equal(t, tt.want, formatted.SchemaRefsSource)

			for _, format := range []formatter.FormatType{formatter.SHORT, formatter.TEXT, formatter.JSON, formatter.YAML} {
				var output bytes.Buffer
				require.NoError(t, formatter.Format(&output, []formattedAgentRemoteConfig{formatted}, format))
				assert.Contains(t, output.String(), "schemaRefs")
				assert.Contains(t, output.String(), "schemaRefsSource")
				assert.Contains(t, output.String(), tt.want)

				for _, ref := range tt.refs {
					assert.Contains(t, output.String(), ref)
				}
			}
		})
	}
}
