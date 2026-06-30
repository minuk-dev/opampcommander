package admin_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	adminsvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/admin"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

// mockConnectionUsecase is a mock implementation of agentport.ConnectionUsecase.
// Only ListConnections is exercised by the admin service; the rest satisfy the interface.
type mockConnectionUsecase struct {
	mock.Mock
}

func (m *mockConnectionUsecase) ListConnections(
	ctx context.Context, namespace string, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Connection], error) {
	args := m.Called(ctx, namespace, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Connection])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockConnectionUsecase) GetConnectionByInstanceUID(
	ctx context.Context, instanceUID uuid.UUID,
) (*agentmodel.Connection, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	conn, ok := args.Get(0).(*agentmodel.Connection)
	if !ok {
		return nil, errMock
	}

	return conn, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockConnectionUsecase) GetOrCreateConnectionByID(
	ctx context.Context, id any,
) (*agentmodel.Connection, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	conn, ok := args.Get(0).(*agentmodel.Connection)
	if !ok {
		return nil, errMock
	}

	return conn, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockConnectionUsecase) GetConnectionByID(ctx context.Context, id any) (*agentmodel.Connection, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	conn, ok := args.Get(0).(*agentmodel.Connection)
	if !ok {
		return nil, errMock
	}

	return conn, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockConnectionUsecase) SaveConnection(ctx context.Context, connection *agentmodel.Connection) error {
	args := m.Called(ctx, connection)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockConnectionUsecase) DeleteConnection(ctx context.Context, connection *agentmodel.Connection) error {
	args := m.Called(ctx, connection)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockConnectionUsecase) SendServerToAgent(
	ctx context.Context, instanceUID uuid.UUID, message *protobufs.ServerToAgent,
) error {
	args := m.Called(ctx, instanceUID, message)

	return args.Error(0) //nolint:wrapcheck // mock error
}

// mockClusterConnectionUsecase is a mock implementation of agentport.ClusterConnectionUsecase.
type mockClusterConnectionUsecase struct {
	mock.Mock
}

func (m *mockClusterConnectionUsecase) ListClusterConnections(
	ctx context.Context, namespace string, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.ServerConnection], error) {
	args := m.Called(ctx, namespace, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.ServerConnection])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func newSvc(t *testing.T, conn *mockConnectionUsecase, cluster *mockClusterConnectionUsecase) *adminsvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	// agentUsecase and agentNotificationUsecase are unused by the listing methods under test.
	return adminsvc.New(nil, conn, cluster, nil, base.Logger)
}

func TestService_ListConnections(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockConn := new(mockConnectionUsecase)
		svc := newSvc(t, mockConn, new(mockClusterConnectionUsecase))

		conn := &agentmodel.Connection{
			UID:                uuid.New(),
			InstanceUID:        uuid.New(),
			Namespace:          "default",
			Type:               agentmodel.ConnectionTypeWebSocket,
			LastCommunicatedAt: time.Now(),
		}
		opts := &applicationport.ListOptions{Limit: 10}
		resp := &model.ListResponse[*agentmodel.Connection]{Items: []*agentmodel.Connection{conn}, Continue: "next"}
		mockConn.On("ListConnections", ctx, "default", opts.ToDomain()).Return(resp, nil)

		result, err := svc.ListConnections(ctx, "default", opts)

		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, conn.UID, result.Items[0].ID)
		assert.Equal(t, "default", result.Items[0].Namespace)
		// Node-local connections carry no owning ServerID.
		assert.Empty(t, result.Items[0].ServerID)
		assert.True(t, result.Items[0].Alive)
		assert.Equal(t, "next", result.Metadata.Continue)
		mockConn.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockConn := new(mockConnectionUsecase)
		svc := newSvc(t, mockConn, new(mockClusterConnectionUsecase))

		opts := &applicationport.ListOptions{Limit: 10}
		mockConn.On("ListConnections", ctx, "default", opts.ToDomain()).Return(nil, errMock)

		result, err := svc.ListConnections(ctx, "default", opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to list connections")
		mockConn.AssertExpectations(t)
	})
}

func TestService_ListClusterConnections(t *testing.T) {
	t.Parallel()

	t.Run("success carries owning ServerID", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockCluster := new(mockClusterConnectionUsecase)
		svc := newSvc(t, new(mockConnectionUsecase), mockCluster)

		serverConn := &agentmodel.ServerConnection{
			ServerID:           "server-a",
			UID:                uuid.New(),
			InstanceUID:        uuid.New(),
			Namespace:          "default",
			Type:               agentmodel.ConnectionTypeWebSocket,
			LastCommunicatedAt: time.Now(),
		}
		opts := &applicationport.ListOptions{Limit: 10}
		resp := &model.ListResponse[*agentmodel.ServerConnection]{
			Items:    []*agentmodel.ServerConnection{serverConn},
			Continue: "next",
		}
		mockCluster.On("ListClusterConnections", ctx, "default", opts.ToDomain()).Return(resp, nil)

		result, err := svc.ListClusterConnections(ctx, "default", opts)

		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "server-a", result.Items[0].ServerID)
		assert.True(t, result.Items[0].Alive)
		mockCluster.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockCluster := new(mockClusterConnectionUsecase)
		svc := newSvc(t, new(mockConnectionUsecase), mockCluster)

		opts := &applicationport.ListOptions{Limit: 10}
		mockCluster.On("ListClusterConnections", ctx, "default", opts.ToDomain()).Return(nil, errMock)

		result, err := svc.ListClusterConnections(ctx, "default", opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to list cluster connections")
		mockCluster.AssertExpectations(t)
	})
}
