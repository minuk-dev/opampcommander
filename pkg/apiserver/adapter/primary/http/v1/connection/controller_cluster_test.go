package connection_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/connection"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// TestConnectionController_ListCluster covers the cluster-scoped branch of List, which
// dispatches to ListClusterConnections when scope=cluster or a serverId is supplied.
func TestConnectionController_ListCluster(t *testing.T) {
	t.Parallel()

	t.Run("scope=cluster dispatches to ListClusterConnections", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		adminUsecase := newMockAdminUsecase(t)
		controller := connection.NewController(ctrlBase.Logger, adminUsecase)
		ctrlBase.SetupRouter(controller)

		adminUsecase.On("ListClusterConnections", mock.Anything, "default", "", mock.Anything).
			Return(v1.NewConnectionListResponse(nil, v1.ListMeta{RemainingItemCount: 0, Continue: ""}), nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet, "/api/v1/namespaces/default/connections?scope=cluster", nil,
		)
		require.NoError(t, err)
		ctrlBase.Router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("a serverId implies cluster scope", func(t *testing.T) {
		t.Parallel()

		ctrlBase := testutil.NewBase(t).ForController()
		adminUsecase := newMockAdminUsecase(t)
		controller := connection.NewController(ctrlBase.Logger, adminUsecase)
		ctrlBase.SetupRouter(controller)

		adminUsecase.On("ListClusterConnections", mock.Anything, "default", "server-1", mock.Anything).
			Return(v1.NewConnectionListResponse(nil, v1.ListMeta{RemainingItemCount: 0, Continue: ""}), nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet, "/api/v1/namespaces/default/connections?serverId=server-1", nil,
		)
		require.NoError(t, err)
		ctrlBase.Router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusOK, recorder.Code)
	})
}
