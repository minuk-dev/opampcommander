package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

func TestAgentRemoteConfigResourceEntity_SchemaRefsSourceRoundTrip(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		source agentmodel.SchemaRefsSource
	}{
		{name: "legacy"},
		{name: "automatic", source: agentmodel.SchemaRefsSourceAuto},
		{name: "explicit", source: agentmodel.SchemaRefsSourceExplicit},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := &agentmodel.AgentRemoteConfig{
				Spec:   agentmodel.AgentRemoteConfigSpec{SchemaRefs: []string{"contrib"}},
				Status: agentmodel.AgentRemoteConfigResourceStatus{SchemaRefsSource: tt.source},
			}
			data, err := bson.Marshal(entity.AgentRemoteConfigResourceEntityFromDomain(config))
			require.NoError(t, err)

			var stored entity.AgentRemoteConfigResourceEntity
			require.NoError(t, bson.Unmarshal(data, &stored))
			assert.Equal(t, tt.source, stored.ToDomain().Status.SchemaRefsSource)
			assert.Equal(t, config.Spec.SchemaRefs, stored.ToDomain().Spec.SchemaRefs)

			_, lookupErr := bson.Raw(data).LookupErr("status", "schemaRefsSource")
			if tt.source == "" {
				require.Error(t, lookupErr)
			} else {
				require.NoError(t, lookupErr)
			}
		})
	}
}
