//nolint:testpackage // white-box test of unexported instanceUID conflict helpers
package opamp

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	domainport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// stubAgentUsecase satisfies agentport.AgentUsecase by embedding the interface; only the
// methods used by detectInstanceUIDConflict / recordInstanceUIDConflict are overridden.
type stubAgentUsecase struct {
	agentport.AgentUsecase

	getResult *agentmodel.Agent
	getErr    error
	saved     *agentmodel.Agent
}

func (s *stubAgentUsecase) GetAgent(_ context.Context, _ uuid.UUID) (*agentmodel.Agent, error) {
	return s.getResult, s.getErr
}

func (s *stubAgentUsecase) SaveAgent(_ context.Context, a *agentmodel.Agent) error {
	s.saved = a

	return nil
}

// stubConnectionUsecase satisfies agentport.ConnectionUsecase the same way.
type stubConnectionUsecase struct {
	agentport.ConnectionUsecase

	byInstanceUID    *agentmodel.Connection
	byInstanceUIDErr error
	byID             *agentmodel.Connection
	byIDErr          error
}

func (s *stubConnectionUsecase) GetConnectionByInstanceUID(
	_ context.Context,
	_ uuid.UUID,
) (*agentmodel.Connection, error) {
	return s.byInstanceUID, s.byInstanceUIDErr
}

func (s *stubConnectionUsecase) GetConnectionByID(_ context.Context, _ any) (*agentmodel.Connection, error) {
	return s.byID, s.byIDErr
}

func newTestService(t *testing.T, agentUC agentport.AgentUsecase, connUC agentport.ConnectionUsecase) *Service {
	t.Helper()

	return &Service{
		clock:             clock.NewRealClock(),
		logger:            slog.New(slog.DiscardHandler),
		agentUsecase:      agentUC,
		connectionUsecase: connUC,
	}
}

func aliveConn(t *testing.T, uid uuid.UUID, instanceUID uuid.UUID) *agentmodel.Connection {
	t.Helper()

	return &agentmodel.Connection{
		ID:                 struct{}{},
		Type:               agentmodel.ConnectionTypeWebSocket,
		UID:                uid,
		InstanceUID:        instanceUID,
		LastCommunicatedAt: time.Now(),
	}
}

func TestCreateRenewalServerToAgent(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	oldUID := uuid.New()
	newUID := uuid.New()
	got := svc.createRenewalServerToAgent(oldUID, newUID)

	require.NotNil(t, got)
	assert.Equal(t, oldUID[:], got.GetInstanceUid())
	require.NotNil(t, got.GetAgentIdentification())
	assert.Equal(t, newUID[:], got.GetAgentIdentification().GetNewInstanceUid())
}

func TestNewInstanceUIDGeneratesV7(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, nil, nil)
	id := svc.newInstanceUID(svc.logger)

	assert.NotEqual(t, uuid.Nil, id)
	assert.Equal(t, uuid.Version(7), id.Version())
}

func TestDetectInstanceUIDConflict_NoConflictForBrandNewAgent(t *testing.T) {
	t.Parallel()

	agentUC := &stubAgentUsecase{getErr: domainport.ErrResourceNotExist}
	connUC := &stubConnectionUsecase{byInstanceUIDErr: agentport.ErrConnectionNotFound}

	svc := newTestService(t, agentUC, connUC)

	conflict := svc.detectInstanceUIDConflict(t.Context(), nil, uuid.New(), &protobufs.AgentToServer{})

	assert.Nil(t, conflict)
}

func TestDetectInstanceUIDConflict_NilInstanceUID(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &stubAgentUsecase{}, &stubConnectionUsecase{})

	conflict := svc.detectInstanceUIDConflict(t.Context(), nil, uuid.Nil, &protobufs.AgentToServer{})

	assert.Nil(t, conflict)
}

func TestDetectInstanceUIDConflict_LiveConnectionAloneIsNotConflict(t *testing.T) {
	t.Parallel()

	// A different live connection on the same instanceUID by itself is not a conflict —
	// it happens on every WebSocket reconnect before OnConnectionClose cleanup runs.
	// We must only flag a conflict when the identifying attributes also differ.
	instanceUID := uuid.New()
	other := aliveConn(t, uuid.New(), instanceUID)
	current := aliveConn(t, uuid.New(), instanceUID)

	agentUC := &stubAgentUsecase{getErr: domainport.ErrResourceNotExist}
	connUC := &stubConnectionUsecase{byInstanceUID: other, byID: current}

	svc := newTestService(t, agentUC, connUC)

	conflict := svc.detectInstanceUIDConflict(t.Context(), nil, instanceUID, &protobufs.AgentToServer{})

	assert.Nil(t, conflict)
}

func TestDetectInstanceUIDConflict_IdentifyingAttrsMismatch(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()

	stored := agentmodel.NewAgent(instanceUID, agentmodel.WithDescription(&agent.Description{
		IdentifyingAttributes: map[string]string{
			"service.instance.id": "stored-instance",
			"service.name":        "collector",
		},
	}))

	agentUC := &stubAgentUsecase{getResult: stored}
	connUC := &stubConnectionUsecase{byInstanceUIDErr: agentport.ErrConnectionNotFound}

	svc := newTestService(t, agentUC, connUC)

	msg := &protobufs.AgentToServer{
		AgentDescription: &protobufs.AgentDescription{
			IdentifyingAttributes: []*protobufs.KeyValue{
				{Key: "service.instance.id", Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{StringValue: "different-instance"},
				}},
				{Key: "service.name", Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{StringValue: "collector"},
				}},
			},
		},
	}

	conflict := svc.detectInstanceUIDConflict(t.Context(), nil, instanceUID, msg)

	require.NotNil(t, conflict)
	assert.Equal(t, conflictReasonIdentifyingAttrs, conflict.reason)
	assert.Same(t, stored, conflict.existingAgent)
}

func TestDetectInstanceUIDConflict_IdentifyingAttrsMatchNoConflict(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()
	stored := agentmodel.NewAgent(instanceUID, agentmodel.WithDescription(&agent.Description{
		IdentifyingAttributes: map[string]string{"service.instance.id": "same"},
	}))

	agentUC := &stubAgentUsecase{getResult: stored}
	connUC := &stubConnectionUsecase{byInstanceUIDErr: agentport.ErrConnectionNotFound}

	svc := newTestService(t, agentUC, connUC)

	msg := &protobufs.AgentToServer{
		AgentDescription: &protobufs.AgentDescription{
			IdentifyingAttributes: []*protobufs.KeyValue{
				{Key: "service.instance.id", Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{StringValue: "same"},
				}},
			},
		},
	}

	conflict := svc.detectInstanceUIDConflict(t.Context(), nil, instanceUID, msg)

	assert.Nil(t, conflict)
}

func TestDetectInstanceUIDConflict_LiveAndIdentifyingCombined(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()
	stored := agentmodel.NewAgent(instanceUID, agentmodel.WithDescription(&agent.Description{
		IdentifyingAttributes: map[string]string{"service.instance.id": "stored"},
	}))

	other := aliveConn(t, uuid.New(), instanceUID)
	current := aliveConn(t, uuid.New(), instanceUID)

	agentUC := &stubAgentUsecase{getResult: stored}
	connUC := &stubConnectionUsecase{byInstanceUID: other, byID: current}

	svc := newTestService(t, agentUC, connUC)

	msg := &protobufs.AgentToServer{
		AgentDescription: &protobufs.AgentDescription{
			IdentifyingAttributes: []*protobufs.KeyValue{
				{Key: "service.instance.id", Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{StringValue: "different"},
				}},
			},
		},
	}

	conflict := svc.detectInstanceUIDConflict(t.Context(), nil, instanceUID, msg)

	require.NotNil(t, conflict)
	assert.Equal(t, conflictReasonLiveAndIdentifying, conflict.reason)
}

func TestRecordInstanceUIDConflict_AddsConditionAndSaves(t *testing.T) {
	t.Parallel()

	stored := agentmodel.NewAgent(uuid.New())
	agentUC := &stubAgentUsecase{}
	svc := newTestService(t, agentUC, &stubConnectionUsecase{})

	oldUID := uuid.New()
	newUID := uuid.New()

	svc.recordInstanceUIDConflict(t.Context(), svc.logger,
		&instanceUIDConflict{reason: conflictReasonIdentifyingAttrs, existingAgent: stored},
		oldUID, newUID)

	assert.Same(t, stored, agentUC.saved)
	condition := stored.GetCondition(agentmodel.AgentConditionTypeInstanceUIDConflict)
	require.NotNil(t, condition)
	assert.Equal(t, agentmodel.AgentConditionStatusTrue, condition.Status)
	assert.Contains(t, condition.Message, oldUID.String())
	assert.Contains(t, condition.Message, newUID.String())
	assert.Contains(t, condition.Message, conflictReasonIdentifyingAttrs)
}

func TestRecordInstanceUIDConflict_NoExistingAgentSkipsSave(t *testing.T) {
	t.Parallel()

	agentUC := &stubAgentUsecase{}
	svc := newTestService(t, agentUC, &stubConnectionUsecase{})

	svc.recordInstanceUIDConflict(t.Context(), svc.logger,
		&instanceUIDConflict{reason: conflictReasonIdentifyingAttrs, existingAgent: nil},
		uuid.New(), uuid.New())

	assert.Nil(t, agentUC.saved)
}

func TestRecordInstanceUIDConflict_ReplacesPreviousAudit(t *testing.T) {
	t.Parallel()

	stored := agentmodel.NewAgent(uuid.New())
	svc := newTestService(t, &stubAgentUsecase{}, &stubConnectionUsecase{})

	firstNew := uuid.New()
	secondNew := uuid.New()
	oldUID := uuid.New()

	svc.recordInstanceUIDConflict(t.Context(), svc.logger,
		&instanceUIDConflict{reason: conflictReasonLiveAndIdentifying, existingAgent: stored},
		oldUID, firstNew)
	svc.recordInstanceUIDConflict(t.Context(), svc.logger,
		&instanceUIDConflict{reason: conflictReasonIdentifyingAttrs, existingAgent: stored},
		oldUID, secondNew)

	condition := stored.GetCondition(agentmodel.AgentConditionTypeInstanceUIDConflict)
	require.NotNil(t, condition)
	assert.Contains(t, condition.Message, secondNew.String())
	assert.Contains(t, condition.Message, conflictReasonIdentifyingAttrs)
	assert.NotContains(t, condition.Message, firstNew.String())

	count := 0

	for _, c := range stored.Status.Conditions {
		if c.Type == agentmodel.AgentConditionTypeInstanceUIDConflict {
			count++
		}
	}

	assert.Equal(t, 1, count, "should not duplicate the conflict condition")
}

var errTestTransient = errors.New("transient persistence failure")

func TestDetectInstanceUIDConflict_AgentFetchTransientErrorIsNotConflict(t *testing.T) {
	t.Parallel()

	agentUC := &stubAgentUsecase{getErr: errTestTransient}
	connUC := &stubConnectionUsecase{byInstanceUIDErr: agentport.ErrConnectionNotFound}

	svc := newTestService(t, agentUC, connUC)

	msg := &protobufs.AgentToServer{
		AgentDescription: &protobufs.AgentDescription{
			IdentifyingAttributes: []*protobufs.KeyValue{
				{Key: "service.instance.id", Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{StringValue: "x"},
				}},
			},
		},
	}

	conflict := svc.detectInstanceUIDConflict(t.Context(), nil, uuid.New(), msg)

	assert.Nil(t, conflict)
}

func TestHandleInstanceUIDConflict_ReturnsRenewalAndAudits(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()

	stored := agentmodel.NewAgent(instanceUID, agentmodel.WithDescription(&agent.Description{
		IdentifyingAttributes: map[string]string{serviceInstanceIDAttribute: "stored"},
	}))

	agentUC := &stubAgentUsecase{getResult: stored}
	connUC := &stubConnectionUsecase{byInstanceUIDErr: agentport.ErrConnectionNotFound}

	svc := newTestService(t, agentUC, connUC)

	msg := &protobufs.AgentToServer{
		AgentDescription: &protobufs.AgentDescription{
			IdentifyingAttributes: []*protobufs.KeyValue{
				{Key: serviceInstanceIDAttribute, Value: &protobufs.AnyValue{
					Value: &protobufs.AnyValue_StringValue{StringValue: "different"},
				}},
			},
		},
	}

	response := svc.handleInstanceUIDConflict(t.Context(), svc.logger, nil, instanceUID, msg)

	require.NotNil(t, response)
	assert.Equal(t, instanceUID[:], response.GetInstanceUid())
	require.NotNil(t, response.GetAgentIdentification())
	newUIDBytes := response.GetAgentIdentification().GetNewInstanceUid()
	require.Len(t, newUIDBytes, 16)
	newUID, err := uuid.FromBytes(newUIDBytes)
	require.NoError(t, err)
	assert.NotEqual(t, instanceUID, newUID)

	assert.Same(t, stored, agentUC.saved)
	cond := stored.GetCondition(agentmodel.AgentConditionTypeInstanceUIDConflict)
	require.NotNil(t, cond)
	assert.Contains(t, cond.Message, newUID.String())
}

func TestHandleInstanceUIDConflict_NoConflictReturnsNil(t *testing.T) {
	t.Parallel()

	agentUC := &stubAgentUsecase{getErr: domainport.ErrResourceNotExist}
	connUC := &stubConnectionUsecase{byInstanceUIDErr: agentport.ErrConnectionNotFound}

	svc := newTestService(t, agentUC, connUC)

	response := svc.handleInstanceUIDConflict(t.Context(), svc.logger, nil, uuid.New(), &protobufs.AgentToServer{})

	assert.Nil(t, response)
}
