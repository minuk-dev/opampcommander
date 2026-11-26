package primary_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx" //nolint:depguard // Verify the gateway is constructible without legacy HTTP dependencies.
	"go.uber.org/fx/fxtest"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/adapter/primary"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/observability"
)

type gatewayUsecase struct{ usecase.OpAMPUsecase }

func TestGatewayHTTPRoutes(t *testing.T) {
	t.Parallel()

	var engine *gin.Engine

	_ = fxtest.New(t,
		primary.NewHTTP(true),
		fx.Supply(&config.ServerSettings{}, &observability.Service{}, slog.Default()),
		fx.Provide(func() usecase.OpAMPUsecase { return &gatewayUsecase{} }),
		fx.Populate(&engine),
	)
	require.NotNil(t, engine)

	registered := engine.Routes()

	routes := make([]string, 0, len(registered))
	for _, route := range registered {
		routes = append(routes, route.Method+" "+route.Path)
	}

	require.ElementsMatch(t, []string{"GET /api/v1/opamp", "POST /api/v1/opamp"}, routes)

	for _, path := range []string{
		"/api/v1/namespaces", "/api/v1/users", "/api/v1/roles", "/api/runtime/v1/namespaces/default", "/docs",
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer legacy-token")
		engine.ServeHTTP(response, request)
		require.Equal(t, http.StatusNotFound, response.Code, path)
	}
}
