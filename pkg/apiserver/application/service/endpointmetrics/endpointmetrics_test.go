package endpointmetrics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	endpointmetricssvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/endpointmetrics"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

const testDefaultWindow = time.Minute

// mockEndpointMetricsUsecase is a mock implementation of agentport.EndpointMetricsUsecase.
type mockEndpointMetricsUsecase struct {
	mock.Mock
}

func (m *mockEndpointMetricsUsecase) GetEndpointThroughput(
	ctx context.Context, namespace, name string, window time.Duration, evaluatedAt time.Time,
) (*agentmodel.EndpointThroughput, error) {
	args := m.Called(ctx, namespace, name, window, evaluatedAt)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	tp, ok := args.Get(0).(*agentmodel.EndpointThroughput)
	if !ok {
		return nil, errMock
	}

	return tp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockEndpointMetricsUsecase) ListEndpointThroughput(
	ctx context.Context, namespace string, window time.Duration, evaluatedAt time.Time,
) ([]*agentmodel.EndpointThroughput, error) {
	args := m.Called(ctx, namespace, window, evaluatedAt)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	tps, ok := args.Get(0).([]*agentmodel.EndpointThroughput)
	if !ok {
		return nil, errMock
	}

	return tps, args.Error(1) //nolint:wrapcheck // mock error
}

func newSvc(t *testing.T, m *mockEndpointMetricsUsecase) *endpointmetricssvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return endpointmetricssvc.NewEndpointMetricsService(m, testDefaultWindow, base.Logger)
}

func newThroughput(namespace, name string, window time.Duration) *agentmodel.EndpointThroughput {
	return &agentmodel.EndpointThroughput{
		Namespace:   namespace,
		Name:        name,
		EvaluatedAt: time.Now(),
		Window:      window,
	}
}

func TestService_GetEndpointThroughput(t *testing.T) {
	t.Parallel()

	t.Run("uses supplied window", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUC := new(mockEndpointMetricsUsecase)
		svc := newSvc(t, mockUC)

		window := 10 * time.Minute
		mockUC.On("GetEndpointThroughput", ctx, "default", "ep-1", window, mock.AnythingOfType("time.Time")).
			Return(newThroughput("default", "ep-1", window), nil)

		result, err := svc.GetEndpointThroughput(ctx, "default", "ep-1", window)

		require.NoError(t, err)
		assert.Equal(t, "ep-1", result.Name)
		mockUC.AssertExpectations(t)
	})

	t.Run("falls back to default window when window non-positive", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUC := new(mockEndpointMetricsUsecase)
		svc := newSvc(t, mockUC)

		mockUC.On("GetEndpointThroughput", ctx, "default", "ep-1", testDefaultWindow, mock.AnythingOfType("time.Time")).
			Return(newThroughput("default", "ep-1", testDefaultWindow), nil)

		_, err := svc.GetEndpointThroughput(ctx, "default", "ep-1", 0)

		require.NoError(t, err)
		mockUC.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUC := new(mockEndpointMetricsUsecase)
		svc := newSvc(t, mockUC)

		mockUC.On("GetEndpointThroughput", ctx, "default", "ep-1", testDefaultWindow, mock.AnythingOfType("time.Time")).
			Return(nil, errMock)

		result, err := svc.GetEndpointThroughput(ctx, "default", "ep-1", 0)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "get endpoint throughput")
		mockUC.AssertExpectations(t)
	})
}

func TestService_ListEndpointThroughput(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUC := new(mockEndpointMetricsUsecase)
		svc := newSvc(t, mockUC)

		mockUC.On("ListEndpointThroughput", ctx, "default", testDefaultWindow, mock.AnythingOfType("time.Time")).
			Return([]*agentmodel.EndpointThroughput{newThroughput("default", "ep-1", testDefaultWindow)}, nil)

		result, err := svc.ListEndpointThroughput(ctx, "default", 0)

		require.NoError(t, err)
		assert.Equal(t, v1.EndpointThroughputKind, result.Kind)
		assert.Len(t, result.Items, 1)
		mockUC.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockUC := new(mockEndpointMetricsUsecase)
		svc := newSvc(t, mockUC)

		mockUC.On("ListEndpointThroughput", ctx, "default", testDefaultWindow, mock.AnythingOfType("time.Time")).
			Return(nil, errMock)

		result, err := svc.ListEndpointThroughput(ctx, "default", 0)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "list endpoint throughput")
		mockUC.AssertExpectations(t)
	})
}
