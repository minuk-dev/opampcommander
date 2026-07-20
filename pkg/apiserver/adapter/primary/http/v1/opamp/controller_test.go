package opamp_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/open-telemetry/opamp-go/protobufs"
	opamptypes "github.com/open-telemetry/opamp-go/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/opamp"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	goleak.VerifyTestMain(m)
}

// spyUsecase is a no-op OpAMPUsecase that records OnConnectedWithType calls. A plain spy is
// used instead of testify so the opamp-go server can drive callbacks freely during Handle
// without tripping strict expectations.
type spyUsecase struct {
	onConnectedWithTypeCalls int
	lastIsWebSocket          bool
}

func (s *spyUsecase) OnConnected(_ context.Context, _ opamptypes.Connection) {}

func (s *spyUsecase) OnConnectedWithType(_ context.Context, _ opamptypes.Connection, isWebSocket bool) {
	s.onConnectedWithTypeCalls++
	s.lastIsWebSocket = isWebSocket
}

func (s *spyUsecase) OnMessage(
	_ context.Context, _ opamptypes.Connection, _ *protobufs.AgentToServer,
) *protobufs.ServerToAgent {
	return nil
}

func (s *spyUsecase) OnConnectionClose(_ opamptypes.Connection) {}

func (s *spyUsecase) OnReadMessageError(_ opamptypes.Connection, _ int, _ []byte, _ error) {}

func (s *spyUsecase) OnMessageResponseError(_ opamptypes.Connection, _ *protobufs.ServerToAgent, _ error) {
}

func TestController_New_RoutesInfo(t *testing.T) {
	t.Parallel()

	controller := opamp.NewController(&spyUsecase{}, slog.Default())
	require.NotNil(t, controller)

	routes := controller.RoutesInfo()
	require.Len(t, routes, 2)

	got := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		got[route.Method+" "+route.Path] = struct{}{}

		assert.NotNil(t, route.HandlerFunc)
	}

	assert.Contains(t, got, "GET /api/v1/opamp")
	assert.Contains(t, got, "POST /api/v1/opamp")
}

func TestController_OnConnecting(t *testing.T) {
	t.Parallel()

	t.Run("websocket connection is accepted and marked as websocket", func(t *testing.T) {
		t.Parallel()

		spy := &spyUsecase{}
		controller := opamp.NewController(spy, slog.Default())

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/opamp", nil)
		require.NoError(t, err)
		req.Header.Set("Upgrade", "websocket")

		resp := controller.OnConnecting(req)
		assert.True(t, resp.Accept)
		assert.Equal(t, http.StatusOK, resp.HTTPStatusCode)

		// Invoke the OnConnected callback to exercise the closure body.
		resp.ConnectionCallbacks.OnConnected(t.Context(), nil)
		assert.Equal(t, 1, spy.onConnectedWithTypeCalls)
		assert.True(t, spy.lastIsWebSocket)
	})

	t.Run("plain http connection is not marked as websocket", func(t *testing.T) {
		t.Parallel()

		spy := &spyUsecase{}
		controller := opamp.NewController(spy, slog.Default())

		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/opamp", nil)
		require.NoError(t, err)

		resp := controller.OnConnecting(req)
		assert.True(t, resp.Accept)

		resp.ConnectionCallbacks.OnConnected(t.Context(), nil)
		assert.Equal(t, 1, spy.onConnectedWithTypeCalls)
		assert.False(t, spy.lastIsWebSocket)
	})
}

func TestController_Handle(t *testing.T) {
	t.Parallel()

	ctrlBase := testutil.NewBase(t).ForController()
	controller := opamp.NewController(&spyUsecase{}, slog.Default())
	ctrlBase.SetupRouter(controller)

	// A GET without a websocket upgrade is rejected by the opamp-go handler; the point is that
	// Handle delegates to the attached handler and produces a response without panicking.
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/opamp", nil)
	require.NoError(t, err)
	ctrlBase.Router.ServeHTTP(recorder, req)

	assert.NotEqual(t, http.StatusNotFound, recorder.Code)
}
