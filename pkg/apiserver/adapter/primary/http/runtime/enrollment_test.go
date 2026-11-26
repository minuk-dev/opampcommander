package runtime_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	runtimeapi "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/runtime"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

type enrollmentStore struct {
	port.AgentEnrollmentStore
	key port.AgentEnrollmentKey
}

func (s *enrollmentStore) Lookup(_ context.Context, key port.AgentEnrollmentKey) (string, error) {
	s.key = key

	return "collector-uid", nil
}

func TestEnrollmentRead(t *testing.T) {
	t.Parallel()

	const instance = "abc115aa-730b-4c1d-8638-e2b64eb27b34"
	const path = "/api/runtime/v1/namespaces/tenant/agent-enrollments/" + instance

	config, client := fixture()
	store := &enrollmentStore{}

	controller, err := runtimeapi.NewController(config, client, store)
	require.NoError(t, err)

	engine := gin.New()
	controller.Register(engine)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, port.AgentEnrollmentKey{
		InstallationUID: config.InstallationUID, NamespaceUID: "namespace-uid", InstanceUID: instance,
	}, store.key)

	request = httptest.NewRequestWithContext(t.Context(), http.MethodPut, path, nil)
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	require.Equal(t, http.StatusNotFound, response.Code)
}
