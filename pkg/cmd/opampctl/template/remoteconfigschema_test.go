package template_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/template"
)

func TestRemoteConfigSchemaTemplate(t *testing.T) {
	t.Parallel()

	cmd := template.NewCommand(template.CommandOptions{})
	cmd.SetArgs([]string{"remoteconfigschema", "basic", "-o", "json"})

	var output bytes.Buffer
	cmd.SetOut(&output)
	require.NoError(t, cmd.Execute())

	var schema v1.RemoteConfigSchema
	require.NoError(t, json.Unmarshal(output.Bytes(), &schema))
	require.Equal(t, v1.RemoteConfigSchemaKind, schema.Kind)
	require.Equal(t, "otlp", schema.Spec.Components["receivers"]["otlp"].Type)
	require.Equal(t, "debug", schema.Spec.Components["exporters"]["debug"].Type)
}
