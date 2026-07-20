package agentgroup_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agentgroup/usecasemock"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errGapBoom = errors.New("boom")

func gapSetup(t *testing.T) (*testutil.ControllerBase, *usecasemock.MockUsecase) {
	t.Helper()

	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)

	return ctrlBase, usecase
}

func gapGET(t *testing.T, router *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)

	return recorder
}

const gapBase = "/api/v1/namespaces/default/agentgroups"

func TestAgentGroupController_ListValidation(t *testing.T) {
	t.Parallel()

	t.Run("invalid limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest, gapGET(t, ctrlBase.Router, gapBase+"?limit=abc").Code)
	})

	t.Run("invalid includeDeleted", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest, gapGET(t, ctrlBase.Router, gapBase+"?includeDeleted=maybe").Code)
	})
}

func TestAgentGroupController_GetValidation(t *testing.T) {
	t.Parallel()

	ctrlBase, _ := gapSetup(t)

	require.Equal(t, http.StatusBadRequest, gapGET(t, ctrlBase.Router, gapBase+"/grp?includeDeleted=nope").Code)
}

func TestAgentGroupController_ListAgentsByAgentGroup(t *testing.T) {
	t.Parallel()

	t.Run("returns the agents", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		//exhaustruct:ignore
		usecase.On("ListAgentsByAgentGroup", mock.Anything, "default", "grp", mock.Anything).
			Return(&v1.ListResponse[v1.Agent]{Items: []v1.Agent{}}, nil)

		require.Equal(t, http.StatusOK,
			gapGET(t, ctrlBase.Router, gapBase+"/grp/agents?connected=true").Code)
	})

	t.Run("invalid limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest, gapGET(t, ctrlBase.Router, gapBase+"/grp/agents?limit=abc").Code)
	})

	t.Run("invalid connected", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest, gapGET(t, ctrlBase.Router, gapBase+"/grp/agents?connected=maybe").Code)
	})

	t.Run("domain error", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		usecase.On("ListAgentsByAgentGroup", mock.Anything, "default", "grp", mock.Anything).
			Return(nil, model.ErrResourceNotExist)

		require.Equal(t, http.StatusNotFound, gapGET(t, ctrlBase.Router, gapBase+"/grp/agents").Code)
	})
}

func TestAgentGroupController_ListAgentGroupsByAgentGaps(t *testing.T) {
	t.Parallel()

	agentsBase := "/api/v1/namespaces/default/agents"

	t.Run("invalid id", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest,
			gapGET(t, ctrlBase.Router, agentsBase+"/not-a-uuid/agentgroups").Code)
	})

	t.Run("namespace mismatch is reported as not found", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		uid := uuid.New()
		usecase.On("ListAgentGroupsByAgent", mock.Anything, "default", uid).
			Return(nil, applicationport.ErrAgentNamespaceMismatch)

		require.Equal(t, http.StatusNotFound,
			gapGET(t, ctrlBase.Router, agentsBase+"/"+uid.String()+"/agentgroups").Code)
	})

	t.Run("generic error", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		uid := uuid.New()
		usecase.On("ListAgentGroupsByAgent", mock.Anything, "default", uid).Return(nil, errGapBoom)

		require.Equal(t, http.StatusInternalServerError,
			gapGET(t, ctrlBase.Router, agentsBase+"/"+uid.String()+"/agentgroups").Code)
	})
}

// TestAgentGroupController_MissingParams covers the required :namespace / :name / :id validation
// branches across every handler, unreachable through routing since the segments are never empty.
func TestAgentGroupController_MissingParams(t *testing.T) {
	t.Parallel()

	controller := agentgroup.NewController(usecasemock.NewMockUsecase(t), testutil.NewBase(t).Logger)
	ns := gin.Params{{Key: "namespace", Value: "default"}}

	cases := []struct {
		name    string
		handler gin.HandlerFunc
		params  gin.Params
	}{
		{"Get missing namespace", controller.Get, nil},
		{"Get missing name", controller.Get, ns},
		{"Create missing namespace", controller.Create, nil},
		{"Update missing namespace", controller.Update, nil},
		{"Update missing name", controller.Update, ns},
		{"Delete missing namespace", controller.Delete, nil},
		{"Delete missing name", controller.Delete, ns},
		{"ListAgentsByAgentGroup missing namespace", controller.ListAgentsByAgentGroup, nil},
		{"ListAgentsByAgentGroup missing name", controller.ListAgentsByAgentGroup, ns},
		{"ListAgentGroupsByAgent missing namespace", controller.ListAgentGroupsByAgent, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			require.NoError(t, err)

			ginCtx.Request = req
			ginCtx.Params = tc.params

			tc.handler(ginCtx)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}
