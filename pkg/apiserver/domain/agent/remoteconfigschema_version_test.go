package agentmodel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

func schemaFor(binary, version string) *agentmodel.RemoteConfigSchema {
	return &agentmodel.RemoteConfigSchema{
		Metadata: agentmodel.RemoteConfigSchemaMetadata{},
		Spec: agentmodel.RemoteConfigSchemaSpec{
			Binary:     binary,
			Version:    version,
			Components: nil,
		},
		Status: agentmodel.RemoteConfigSchemaStatus{},
	}
}

func TestCompareCollectorVersion(t *testing.T) {
	t.Parallel()

	assert.Equal(t, -1, agentmodel.CompareCollectorVersion("0.110.0", "0.120.0"))
	assert.Equal(t, 1, agentmodel.CompareCollectorVersion("0.120.1", "0.120.0"))
	assert.Equal(t, 0, agentmodel.CompareCollectorVersion("v0.120.0", "0.120.0"))
	// A pre-release suffix does not change the ordering of the release it belongs to.
	assert.Equal(t, 0, agentmodel.CompareCollectorVersion("0.120.0-rc1", "0.120.0"))
}

func TestResolveSchemaForVersion(t *testing.T) {
	t.Parallel()

	schemas := []*agentmodel.RemoteConfigSchema{
		schemaFor("otelcol-contrib", "0.110.0"),
		schemaFor("otelcol-contrib", "0.130.0"),
		schemaFor("otelcol-contrib", "0.157.0"),
		schemaFor("otelcol", "0.157.0"),
	}

	// Exact match.
	resolved := agentmodel.ResolveSchemaForVersion(schemas, "otelcol-contrib", "0.130.0")
	require.NotNil(t, resolved)
	assert.Equal(t, "0.130.0", resolved.Spec.Version)

	// Between two published schemas: the newer one it is a superset of.
	resolved = agentmodel.ResolveSchemaForVersion(schemas, "otelcol-contrib", "0.135.0")
	require.NotNil(t, resolved)
	assert.Equal(t, "0.130.0", resolved.Spec.Version)

	// Newer than every published schema.
	resolved = agentmodel.ResolveSchemaForVersion(schemas, "otelcol-contrib", "0.200.0")
	require.NotNil(t, resolved)
	assert.Equal(t, "0.157.0", resolved.Spec.Version)

	// Older than every published schema: no schema describes it.
	assert.Nil(t, agentmodel.ResolveSchemaForVersion(schemas, "otelcol-contrib", "0.100.0"))

	// A binary the library has no schema for.
	assert.Nil(t, agentmodel.ResolveSchemaForVersion(schemas, "otelcol-k8s", "0.157.0"))
}
