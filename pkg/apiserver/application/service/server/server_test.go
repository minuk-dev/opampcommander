package server_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	serversvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/server"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

// mockServerUsecase is a mock implementation of agentport.ServerUsecase.
type mockServerUsecase struct {
	mock.Mock
}

func (m *mockServerUsecase) GetServer(ctx context.Context, id string) (*agentmodel.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	srv, ok := args.Get(0).(*agentmodel.Server)
	if !ok {
		return nil, errMock
	}

	return srv, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockServerUsecase) ListServers(ctx context.Context) ([]*agentmodel.Server, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	servers, ok := args.Get(0).([]*agentmodel.Server)
	if !ok {
		return nil, errMock
	}

	return servers, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockServerUsecase) SendMessageToServerByServerID(
	ctx context.Context, serverID string, message serverevent.Message,
) error {
	args := m.Called(ctx, serverID, message)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockServerUsecase) SendMessageToServer(
	ctx context.Context, server *agentmodel.Server, message serverevent.Message,
) error {
	args := m.Called(ctx, server, message)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func newSvc(t *testing.T, srv *mockServerUsecase) *serversvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return serversvc.New(srv, base.Logger)
}

func TestService_ListServers(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUsecase := new(mockServerUsecase)
		svc := newSvc(t, mockUsecase)

		heartbeat := time.Now()
		servers := []*agentmodel.Server{
			{
				ID:              "server-a",
				LastHeartbeatAt: heartbeat,
				Conditions: []model.Condition{
					{
						Type:    "Ready",
						Status:  "True",
						Reason:  "Healthy",
						Message: "all good",
					},
				},
			},
		}
		mockUsecase.On("ListServers", ctx).Return(servers, nil)

		result, err := svc.ListServers(ctx)

		require.NoError(t, err)
		assert.Equal(t, v1.ServerKind, result.Kind)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "server-a", result.Items[0].ID)
		require.Len(t, result.Items[0].Conditions, 1)
		assert.Equal(t, v1.ServerConditionType("Ready"), result.Items[0].Conditions[0].Type)
		assert.Equal(t, v1.ServerConditionStatus("True"), result.Items[0].Conditions[0].Status)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("maps empty conditions to nil", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUsecase := new(mockServerUsecase)
		svc := newSvc(t, mockUsecase)

		servers := []*agentmodel.Server{{ID: "server-b", LastHeartbeatAt: time.Now()}}
		mockUsecase.On("ListServers", ctx).Return(servers, nil)

		result, err := svc.ListServers(ctx)

		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Nil(t, result.Items[0].Conditions)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUsecase := new(mockServerUsecase)
		svc := newSvc(t, mockUsecase)

		mockUsecase.On("ListServers", ctx).Return(nil, errMock)

		result, err := svc.ListServers(ctx)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to list servers")
		mockUsecase.AssertExpectations(t)
	})
}
