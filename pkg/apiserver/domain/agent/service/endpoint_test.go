package agentservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	domainport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
)

// epFakePersistence is a minimal in-memory EndpointPersistencePort for the
// lifecycle tests. A nil stored value makes Get report ErrResourceNotExist.
type epFakePersistence struct {
	stored   *agentmodel.Endpoint
	putCalls int
	lastPut  *agentmodel.Endpoint
}

func (f *epFakePersistence) GetEndpoint(
	_ context.Context, _ string, _ string, _ *model.GetOptions,
) (*agentmodel.Endpoint, error) {
	if f.stored == nil {
		return nil, domainport.ErrResourceNotExist
	}

	return f.stored, nil
}

func (f *epFakePersistence) PutEndpoint(
	_ context.Context, endpoint *agentmodel.Endpoint,
) (*agentmodel.Endpoint, error) {
	f.putCalls++
	f.lastPut = endpoint

	return endpoint, nil
}

func (f *epFakePersistence) ListEndpoints(
	_ context.Context, _ string, _ *model.ListOptions,
) (*model.ListResponse[*agentmodel.Endpoint], error) {
	return &model.ListResponse[*agentmodel.Endpoint]{}, nil
}

var _ agentport.EndpointPersistencePort = (*epFakePersistence)(nil)

func newEndpoint(name string) *agentmodel.Endpoint {
	return &agentmodel.Endpoint{
		Metadata: agentmodel.EndpointMetadata{Name: name, Namespace: "default"},
		Spec:     agentmodel.EndpointSpec{URL: "http://old"},
	}
}

func TestEndpointService_CreateEndpoint_RejectsEmptyName(t *testing.T) {
	t.Parallel()

	persistence := &epFakePersistence{}
	svc := agentservice.NewEndpointService(persistence)

	_, err := svc.CreateEndpoint(t.Context(), newEndpoint(""), "tester")

	require.ErrorIs(t, err, domainport.ErrInvalidArgument)
	assert.Zero(t, persistence.putCalls, "an invalid endpoint must not be persisted")
}

func TestEndpointService_CreateEndpoint_RejectsDuplicate(t *testing.T) {
	t.Parallel()

	persistence := &epFakePersistence{stored: newEndpoint("ep")}
	svc := agentservice.NewEndpointService(persistence)

	_, err := svc.CreateEndpoint(t.Context(), newEndpoint("ep"), "tester")

	require.ErrorIs(t, err, domainport.ErrResourceAlreadyExist)
	assert.Zero(t, persistence.putCalls, "a duplicate must not be persisted")
}

func TestEndpointService_CreateEndpoint_Stamps(t *testing.T) {
	t.Parallel()

	persistence := &epFakePersistence{}
	svc := agentservice.NewEndpointService(persistence)

	created, err := svc.CreateEndpoint(t.Context(), newEndpoint("ep"), "tester")

	require.NoError(t, err)
	assert.Equal(t, 1, persistence.putCalls)
	require.NotEmpty(t, created.Status.Conditions, "creation must record a condition")

	cond := created.Status.Conditions[0]
	assert.Equal(t, model.ConditionTypeCreated, cond.Type)
	assert.Equal(t, "tester", cond.Reason, "the acting user must be stamped as the condition reason")
}

func TestEndpointService_UpdateEndpoint_PreservesImmutableFields(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	stored := newEndpoint("ep")
	stored.Metadata.CreatedAt = createdAt
	stored.Status.Conditions = []model.Condition{{Type: model.ConditionTypeCreated}}

	persistence := &epFakePersistence{stored: stored}
	svc := agentservice.NewEndpointService(persistence)

	incoming := newEndpoint("ep")
	incoming.Metadata.CreatedAt = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	incoming.Spec.URL = "http://new"

	updated, err := svc.UpdateEndpoint(t.Context(), "default", "ep", incoming)

	require.NoError(t, err)
	assert.Equal(t, createdAt, updated.Metadata.CreatedAt, "CreatedAt must be preserved from the stored endpoint")
	assert.Equal(t, "http://new", updated.Spec.URL, "mutable spec must be applied")
	assert.NotEmpty(t, updated.Status.Conditions, "existing lifecycle conditions must be preserved")
}
