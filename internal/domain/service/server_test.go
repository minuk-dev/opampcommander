package service_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/internal/domain/service"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

const (
	testServerID = "server-1"
)

var (
	errDatabaseError = errors.New("database error")
)

type testFakeClock struct {
	now time.Time
}

func (c *testFakeClock) Now() time.Time {
	return c.now
}

func (c *testFakeClock) Since(t time.Time) time.Duration {
	return c.now.Sub(t)
}

func (c *testFakeClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.now.Add(d)

	return ch
}

func (c *testFakeClock) NewTimer(_ time.Duration) clock.Timer {
	return nil
}

func (c *testFakeClock) Sleep(_ time.Duration) {
}

func (c *testFakeClock) Tick(_ time.Duration) <-chan time.Time {
	return nil
}

func newTestFakeClock(t time.Time) *testFakeClock {
	return &testFakeClock{now: t}
}

type MockServerPersistencePort struct {
	mock.Mock
}

func (m *MockServerPersistencePort) GetServer(
	ctx context.Context,
	id string,
) (*model.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	server, ok := args.Get(0).(*model.Server)
	if !ok {
		return nil, errUnexpectedType
	}

	return server, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockServerPersistencePort) ListServers(ctx context.Context) ([]*model.Server, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	servers, ok := args.Get(0).([]*model.Server)
	if !ok {
		return nil, errUnexpectedType
	}

	return servers, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockServerPersistencePort) PutServer(ctx context.Context, server *model.Server) error {
	args := m.Called(ctx, server)

	return args.Error(0) //nolint:wrapcheck // mock error
}

type MockServerEventSenderPort struct {
	mock.Mock
}

func (m *MockServerEventSenderPort) SendMessageToServer(
	ctx context.Context,
	serverID string,
	message serverevent.Message,
) error {
	args := m.Called(ctx, serverID, message)

	return args.Error(0) //nolint:wrapcheck // mock error
}

type MockServerEventReceiverPort struct {
	mock.Mock
}

func (m *MockServerEventReceiverPort) StartReceiver(
	ctx context.Context,
	handler port.ReceiveServerEventHandler,
) error {
	args := m.Called(ctx, handler)

	return args.Error(0) //nolint:wrapcheck // mock error
}

type MockConnectionUsecase struct {
	mock.Mock
}

func (m *MockConnectionUsecase) GetConnectionByInstanceUID(
	ctx context.Context,
	instanceUID uuid.UUID,
) (*model.Connection, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	conn, ok := args.Get(0).(*model.Connection)
	if !ok {
		return nil, errUnexpectedType
	}

	return conn, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockConnectionUsecase) GetOrCreateConnectionByID(
	ctx context.Context,
	id any,
) (*model.Connection, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	conn, ok := args.Get(0).(*model.Connection)
	if !ok {
		return nil, errUnexpectedType
	}

	return conn, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockConnectionUsecase) GetConnectionByID(
	ctx context.Context,
	id any,
) (*model.Connection, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	conn, ok := args.Get(0).(*model.Connection)
	if !ok {
		return nil, errUnexpectedType
	}

	return conn, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockConnectionUsecase) ListConnections(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Connection], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*model.Connection])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockConnectionUsecase) SaveConnection(
	ctx context.Context,
	connection *model.Connection,
) error {
	args := m.Called(ctx, connection)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *MockConnectionUsecase) DeleteConnection(
	ctx context.Context,
	connection *model.Connection,
) error {
	args := m.Called(ctx, connection)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *MockConnectionUsecase) SendServerToAgent(
	ctx context.Context,
	instanceUID uuid.UUID,
	message *protobufs.ServerToAgent,
) error {
	args := m.Called(ctx, instanceUID, message)

	return args.Error(0) //nolint:wrapcheck // mock error
}

type MockAgentUsecase struct {
	mock.Mock
}

func (m *MockAgentUsecase) GetAgent(
	ctx context.Context,
	instanceUID uuid.UUID,
) (*model.Agent, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agent, ok := args.Get(0).(*model.Agent)
	if !ok {
		return nil, errUnexpectedType
	}

	return agent, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecase) GetOrCreateAgent(
	ctx context.Context,
	instanceUID uuid.UUID,
) (*model.Agent, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agent, ok := args.Get(0).(*model.Agent)
	if !ok {
		return nil, errUnexpectedType
	}

	return agent, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecase) ListAgentsBySelector(
	ctx context.Context,
	selector model.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	args := m.Called(ctx, selector, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*model.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecase) SaveAgent(ctx context.Context, agent *model.Agent) error {
	args := m.Called(ctx, agent)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecase) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*model.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecase) SearchAgents(
	ctx context.Context,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	args := m.Called(ctx, query, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*model.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func TestServerService_GetServer_CacheHit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverID := testServerID
	now := time.Now()

	mockServer := &model.Server{
		ID:              serverID,
		LastHeartbeatAt: now.Add(-30 * time.Second),
	}

	mockPersistence := new(MockServerPersistencePort)
	mockPersistence.On("GetServer", ctx, serverID).Return(mockServer, nil).Once()

	mockEventSender := new(MockServerEventSenderPort)
	mockEventReceiver := new(MockServerEventReceiverPort)
	mockConnection := new(MockConnectionUsecase)
	mockAgent := new(MockAgentUsecase)

	svc := service.NewServerService(
		slog.Default(),
		mockPersistence,
		mockEventSender,
		mockEventReceiver,
		mockConnection,
		mockAgent,
	)

	fakeClock := newTestFakeClock(now)
	svc.SetClock(fakeClock)

	server1, err := svc.GetServer(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, serverID, server1.ID)

	server2, err := svc.GetServer(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, serverID, server2.ID)

	mockPersistence.AssertExpectations(t)
	mockPersistence.AssertNumberOfCalls(t, "GetServer", 1)
}

func TestServerService_GetServer_CacheMiss_Expired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverID := testServerID
	now := time.Now()

	mockServer := &model.Server{
		ID:              serverID,
		LastHeartbeatAt: now.Add(-2 * time.Minute),
	}

	mockPersistence := new(MockServerPersistencePort)
	mockPersistence.On("GetServer", ctx, serverID).Return(mockServer, nil).Times(2)

	mockEventSender := new(MockServerEventSenderPort)
	mockEventReceiver := new(MockServerEventReceiverPort)
	mockConnection := new(MockConnectionUsecase)
	mockAgent := new(MockAgentUsecase)

	svc := service.NewServerService(
		slog.Default(),
		mockPersistence,
		mockEventSender,
		mockEventReceiver,
		mockConnection,
		mockAgent,
	)

	fakeClock := newTestFakeClock(now)
	svc.SetClock(fakeClock)

	server1, err := svc.GetServer(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, serverID, server1.ID)

	server2, err := svc.GetServer(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, serverID, server2.ID)

	mockPersistence.AssertExpectations(t)
	mockPersistence.AssertNumberOfCalls(t, "GetServer", 2)
}

func TestServerService_GetServer_DatabaseError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverID := testServerID

	mockPersistence := new(MockServerPersistencePort)
	mockPersistence.On("GetServer", ctx, serverID).Return(nil, errDatabaseError)

	mockEventSender := new(MockServerEventSenderPort)
	mockEventReceiver := new(MockServerEventReceiverPort)
	mockConnection := new(MockConnectionUsecase)
	mockAgent := new(MockAgentUsecase)

	svc := service.NewServerService(
		slog.Default(),
		mockPersistence,
		mockEventSender,
		mockEventReceiver,
		mockConnection,
		mockAgent,
	)

	_, err := svc.GetServer(ctx, serverID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	mockPersistence.AssertExpectations(t)
}

func TestServerService_GetServer_CacheUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverID := testServerID
	now := time.Now()

	oldServer := &model.Server{
		ID:              serverID,
		LastHeartbeatAt: now.Add(-30 * time.Second),
	}

	newServer := &model.Server{
		ID:              serverID,
		LastHeartbeatAt: now.Add(-10 * time.Second),
	}

	mockPersistence := new(MockServerPersistencePort)
	mockPersistence.On("GetServer", ctx, serverID).Return(oldServer, nil).Once()
	mockPersistence.On("GetServer", ctx, serverID).Return(newServer, nil).Once()

	mockEventSender := new(MockServerEventSenderPort)
	mockEventReceiver := new(MockServerEventReceiverPort)
	mockConnection := new(MockConnectionUsecase)
	mockAgent := new(MockAgentUsecase)

	svc := service.NewServerService(
		slog.Default(),
		mockPersistence,
		mockEventSender,
		mockEventReceiver,
		mockConnection,
		mockAgent,
	)

	fakeClock := newTestFakeClock(now)
	svc.SetClock(fakeClock)

	server1, err := svc.GetServer(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, oldServer.LastHeartbeatAt, server1.LastHeartbeatAt)

	fakeClock.now = now.Add(2 * time.Minute)

	server2, err := svc.GetServer(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, newServer.LastHeartbeatAt, server2.LastHeartbeatAt)

	mockPersistence.AssertExpectations(t)
	mockPersistence.AssertNumberOfCalls(t, "GetServer", 2)
}
