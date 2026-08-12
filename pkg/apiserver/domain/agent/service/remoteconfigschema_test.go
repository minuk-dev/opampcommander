package agentservice_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

func newSchemaService() *agentservice.RemoteConfigSchemaService {
	return agentservice.NewRemoteConfigSchemaService(inmemory.NewRemoteConfigSchemaRepository())
}

func TestRemoteConfigSchemaService_CRUD(t *testing.T) {
	t.Parallel()

	svc := newSchemaService()
	ctx := t.Context()

	schema := agentmodel.NewRemoteConfigSchema("default", "contrib-0.100", nil, time.Now(), "tester")
	schema.Spec.Binary = "otelcol-contrib"
	schema.Spec.Version = "0.100.0"
	schema.Spec.Components = catalogOf(map[string][]string{"receivers": {"otlp"}})

	created, err := svc.CreateRemoteConfigSchema(ctx, schema, "tester")
	require.NoError(t, err)
	assert.Equal(t, "otelcol-contrib", created.Spec.Binary)

	got, err := svc.GetRemoteConfigSchema(ctx, "default", "contrib-0.100", nil)
	require.NoError(t, err)
	assert.Equal(t, "0.100.0", got.Spec.Version)
	assert.Contains(t, got.Spec.Components["receivers"], "otlp")

	list, err := svc.ListRemoteConfigSchemas(ctx, "default", nil)
	require.NoError(t, err)
	assert.Len(t, list.Items, 1)

	got.Spec.Version = "0.100.1"

	updated, err := svc.UpdateRemoteConfigSchema(ctx, "default", "contrib-0.100", got)
	require.NoError(t, err)
	assert.Equal(t, "0.100.1", updated.Spec.Version)

	err = svc.DeleteRemoteConfigSchema(ctx, "default", "contrib-0.100", time.Now(), "tester")
	require.NoError(t, err)

	_, err = svc.GetRemoteConfigSchema(ctx, "default", "contrib-0.100", nil)
	require.ErrorIs(t, err, model.ErrResourceNotExist)
}

func TestRemoteConfigSchemaService_CreateRejectsDuplicate(t *testing.T) {
	t.Parallel()

	svc := newSchemaService()
	ctx := t.Context()

	schema := agentmodel.NewRemoteConfigSchema("default", "dup", nil, time.Now(), "tester")

	_, err := svc.CreateRemoteConfigSchema(ctx, schema, "tester")
	require.NoError(t, err)

	dup := agentmodel.NewRemoteConfigSchema("default", "dup", nil, time.Now(), "tester")

	_, err = svc.CreateRemoteConfigSchema(ctx, dup, "tester")
	require.ErrorIs(t, err, model.ErrResourceAlreadyExist)
}

func TestRemoteConfigSchemaService_CreateRejectsEmptyName(t *testing.T) {
	t.Parallel()

	svc := newSchemaService()

	schema := agentmodel.NewRemoteConfigSchema("default", "", nil, time.Now(), "tester")

	_, err := svc.CreateRemoteConfigSchema(t.Context(), schema, "tester")
	require.ErrorIs(t, err, model.ErrInvalidArgument)
}
