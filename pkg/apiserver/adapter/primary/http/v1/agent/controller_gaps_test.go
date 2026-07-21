package agent_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agent/usecasemock"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errGapBoom = errors.New("boom")

func gapSetup(t *testing.T) (*testutil.ControllerBase, *usecasemock.MockManageUsecase) {
	t.Helper()

	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockManageUsecase(t)
	controller := agent.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)

	return ctrlBase, usecase
}

func gapReq(t *testing.T, router *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	require.NoError(t, err)

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	router.ServeHTTP(recorder, req)

	return recorder
}

const gapBase = "/api/v1/namespaces/default/agents"

func TestAgentController_ListValidation(t *testing.T) {
	t.Parallel()

	t.Run("invalid connected", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest,
			gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"?connected=maybe", "").Code)
	})

	t.Run("invalid selector", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest,
			gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"?selector=noequalsign", "").Code)
	})

	t.Run("invalid nonIdentifyingSelector", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest,
			gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"?nonIdentifyingSelector=noequalsign", "").Code)
	})

	t.Run("usecase error", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		usecase.On("ListAgents", mock.Anything, "default", mock.Anything).Return(nil, errGapBoom)

		require.Equal(t, http.StatusInternalServerError, gapReq(t, ctrlBase.Router, http.MethodGet, gapBase, "").Code)
	})

	t.Run("empty selector entries are skipped", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		//exhaustruct:ignore
		usecase.On("ListAgents", mock.Anything, "default", mock.Anything).
			Return(&v1.ListResponse[v1.Agent]{Items: []v1.Agent{}}, nil)

		// The empty selector value exercises the "skip empty entry" branch of parseSelector.
		require.Equal(t, http.StatusOK,
			gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"?selector=&selector=env=prod", "").Code)
	})
}

func TestAgentController_SearchValidation(t *testing.T) {
	t.Parallel()

	t.Run("invalid namespace pattern", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest,
			gapReq(t, ctrlBase.Router, http.MethodGet, "/api/v1/namespaces/Invalid_NS/agents/search?q=abc", "").Code)
	})

	t.Run("missing query", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest, gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"/search", "").Code)
	})

	t.Run("invalid query pattern", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest,
			gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"/search?q=bad_query", "").Code)
	})

	t.Run("invalid limit", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest,
			gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"/search?q=abc&limit=xx", "").Code)
	})

	t.Run("invalid connected", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest,
			gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"/search?q=abc&connected=maybe", "").Code)
	})

	t.Run("usecase error", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		usecase.On("SearchAgents", mock.Anything, "default", "abc", mock.Anything).Return(nil, errGapBoom)

		require.Equal(t, http.StatusInternalServerError,
			gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"/search?q=abc", "").Code)
	})
}

func TestAgentController_GetErrors(t *testing.T) {
	t.Parallel()

	t.Run("namespace mismatch is not found", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		uid := uuid.New()
		usecase.On("GetAgent", mock.Anything, "default", uid).Return(nil, applicationport.ErrAgentNamespaceMismatch)

		require.Equal(t, http.StatusNotFound, gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"/"+uid.String(), "").Code)
	})

	t.Run("generic error", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		uid := uuid.New()
		usecase.On("GetAgent", mock.Anything, "default", uid).Return(nil, errGapBoom)

		require.Equal(t, http.StatusInternalServerError,
			gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"/"+uid.String(), "").Code)
	})
}

func TestAgentController_DeleteConnected(t *testing.T) {
	t.Parallel()

	ctrlBase, usecase := gapSetup(t)
	uid := uuid.New()
	usecase.On("DeleteAgent", mock.Anything, "default", uid).Return(applicationport.ErrAgentConnected)

	require.Equal(t, http.StatusConflict, gapReq(t, ctrlBase.Router, http.MethodDelete, gapBase+"/"+uid.String(), "").Code)
}

func TestAgentController_ListEndpoints(t *testing.T) {
	t.Parallel()

	t.Run("returns the endpoints", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		uid := uuid.New()
		//exhaustruct:ignore
		usecase.On("ListAgentEndpoints", mock.Anything, "default", uid).
			Return(&v1.ListResponse[v1.Endpoint]{Items: []v1.Endpoint{}}, nil)

		require.Equal(t, http.StatusOK,
			gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"/"+uid.String()+"/endpoints", "").Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest,
			gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"/not-a-uuid/endpoints", "").Code)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		uid := uuid.New()
		usecase.On("ListAgentEndpoints", mock.Anything, "default", uid).Return(nil, model.ErrResourceNotExist)

		require.Equal(t, http.StatusNotFound,
			gapReq(t, ctrlBase.Router, http.MethodGet, gapBase+"/"+uid.String()+"/endpoints", "").Code)
	})
}

func TestAgentController_Update(t *testing.T) {
	t.Parallel()

	t.Run("updates an agent", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		uid := uuid.New()
		//exhaustruct:ignore
		usecase.On("UpdateAgent", mock.Anything, "default", uid, mock.Anything).Return(&v1.Agent{}, nil)

		require.Equal(t, http.StatusOK,
			gapReq(t, ctrlBase.Router, http.MethodPut, gapBase+"/"+uid.String(), `{}`).Code)
	})

	t.Run("invalid id", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)

		require.Equal(t, http.StatusBadRequest,
			gapReq(t, ctrlBase.Router, http.MethodPut, gapBase+"/not-a-uuid", `{}`).Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		t.Parallel()

		ctrlBase, _ := gapSetup(t)
		uid := uuid.New()

		// A malformed body is a client error: Update wraps the bind error via ginutil.BindJSON
		// (like the other controllers) so HandleValidationError maps it to 400.
		require.Equal(t, http.StatusBadRequest,
			gapReq(t, ctrlBase.Router, http.MethodPut, gapBase+"/"+uid.String(), `{not-json`).Code)
	})

	t.Run("usecase error", func(t *testing.T) {
		t.Parallel()

		ctrlBase, usecase := gapSetup(t)
		uid := uuid.New()
		usecase.On("UpdateAgent", mock.Anything, "default", uid, mock.Anything).Return(nil, errGapBoom)

		require.Equal(t, http.StatusInternalServerError,
			gapReq(t, ctrlBase.Router, http.MethodPut, gapBase+"/"+uid.String(), `{}`).Code)
	})
}

// TestAgentController_MissingNamespace covers the required-:namespace validation branch across
// every handler, unreachable through routing since the segment is never empty when matched.
func TestAgentController_MissingNamespace(t *testing.T) {
	t.Parallel()

	controller := agent.NewController(usecasemock.NewMockManageUsecase(t), testutil.NewBase(t).Logger)

	handlers := map[string]gin.HandlerFunc{
		"List":          controller.List,
		"Search":        controller.Search,
		"Get":           controller.Get,
		"ListEndpoints": controller.ListEndpoints,
		"Update":        controller.Update,
		"Delete":        controller.Delete,
	}

	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			require.NoError(t, err)

			ginCtx.Request = req

			handler(ginCtx)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}
