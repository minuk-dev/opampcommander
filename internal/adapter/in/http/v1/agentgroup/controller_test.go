package agentgroup_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	agentgroupv1 "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agentgroup/usecasemock"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

func TestAgentGroupController_List(t *testing.T) {
	t.Parallel()

	t.Run("List AgentGroups - happycase", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := agentgroup.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		uid1 := uuid.New()
		uid2 := uuid.New()
		groups := []agentgroupv1.AgentGroup{
			{
				UID:        uid1,
				Name:       "g1",
				Attributes: agentgroupv1.Attributes{},
				Selector: agentgroupv1.AgentSelector{
					IdentifyingAttributes:    map[string]string{},
					NonIdentifyingAttributes: map[string]string{},
				},
				CreatedAt: time.Now(),
				CreatedBy: "",
				DeletedAt: nil,
				DeletedBy: nil,
			},
			{
				UID:        uid2,
				Name:       "g2",
				Attributes: agentgroupv1.Attributes{},
				Selector: agentgroupv1.AgentSelector{
					IdentifyingAttributes:    map[string]string{},
					NonIdentifyingAttributes: map[string]string{},
				},
				CreatedAt: time.Now(),
				CreatedBy: "",
				DeletedAt: nil,
				DeletedBy: nil,
			},
		}
		usecase.EXPECT().ListAgentGroups(mock.Anything, mock.Anything).Return(&agentgroupv1.ListResponse{
			Kind:       "AgentGroup",
			APIVersion: "v1",
			Metadata: v1.ListMeta{
				Continue:           "",
				RemainingItemCount: 0,
			},
			Items: groups,
		}, nil)
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentgroups", nil)
		require.NoError(t, err)
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(2), gjson.Get(recorder.Body.String(), "items.#").Int())
		assert.Equal(t, uid1.String(), gjson.Get(recorder.Body.String(), "items.0.uid").String())
		assert.Equal(t, uid2.String(), gjson.Get(recorder.Body.String(), "items.1.uid").String())
	})

	t.Run("List AgentGroups - invalid limit", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := agentgroup.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentgroups?limit=invalid", nil)
		require.NoError(t, err)
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("List AgentGroups - internal error", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := agentgroup.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		usecase.EXPECT().ListAgentGroups(mock.Anything, mock.Anything).Return(nil, assert.AnError)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentgroups", nil)
		require.NoError(t, err)
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestAgentGroupController_Get(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	uid := uuid.New()
	agentGroup := &agentgroupv1.AgentGroup{
		UID:        uid,
		Name:       "g1",
		Attributes: agentgroupv1.Attributes{},
		Selector: agentgroupv1.AgentSelector{
			IdentifyingAttributes:    map[string]string{},
			NonIdentifyingAttributes: map[string]string{},
		},
		CreatedAt: time.Now(),
		CreatedBy: "",
		DeletedAt: nil,
		DeletedBy: nil,
	}
	usecase.EXPECT().GetAgentGroup(mock.Anything, mock.Anything).Return(agentGroup, nil)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentgroups/g1", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, uid.String(), gjson.Get(recorder.Body.String(), "uid").String())
}

func TestAgentGroupController_Get_NotFound(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().GetAgentGroup(mock.Anything, mock.Anything).Return(nil, port.ErrResourceNotExist)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentgroups/notfound", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestAgentGroupController_Get_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().GetAgentGroup(mock.Anything, mock.Anything).Return(nil, assert.AnError)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentgroups/g1", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestAgentGroupController_Create(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	name := "g1"
	body := agentgroupv1.AgentGroup{
		UID:        uuid.New(),
		Name:       name,
		Attributes: agentgroupv1.Attributes{},
		Selector: agentgroupv1.AgentSelector{
			IdentifyingAttributes:    map[string]string{},
			NonIdentifyingAttributes: map[string]string{},
		},
		CreatedAt: time.Now(),
		CreatedBy: "",
		DeletedAt: nil,
		DeletedBy: nil,
	}
	usecase.EXPECT().CreateAgentGroup(mock.Anything, mock.Anything).Return(&body, nil)
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/agentgroups",
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "/api/v1/agentgroups/"+name, recorder.Header().Get("Location"))
}

func TestAgentGroupController_Create_InvalidBody(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/agentgroups",
		strings.NewReader("invalid"),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestAgentGroupController_Create_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	uid := uuid.New()
	payload := agentgroupv1.AgentGroup{
		UID:        uid,
		Name:       "g1",
		Attributes: agentgroupv1.Attributes{},
		Selector: agentgroupv1.AgentSelector{
			IdentifyingAttributes:    map[string]string{},
			NonIdentifyingAttributes: map[string]string{},
		},
		CreatedAt: time.Now(),
		CreatedBy: "",
		DeletedAt: nil,
		DeletedBy: nil,
	}
	usecase.EXPECT().CreateAgentGroup(mock.Anything, mock.Anything).Return(nil, assert.AnError)

	jsonBody, err := json.Marshal(payload)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/agentgroups",
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestAgentGroupController_Update(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	uid := uuid.New()
	group := &agentgroupv1.AgentGroup{
		UID:        uid,
		Name:       "g1",
		Attributes: agentgroupv1.Attributes{},
		Selector: agentgroupv1.AgentSelector{
			IdentifyingAttributes:    map[string]string{},
			NonIdentifyingAttributes: map[string]string{},
		},
		CreatedAt: time.Now(),
		CreatedBy: "",
		DeletedAt: nil,
		DeletedBy: nil,
	}
	usecase.EXPECT().UpdateAgentGroup(mock.Anything, mock.Anything, mock.Anything).Return(group, nil)
	jsonBody, err := json.Marshal(group)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/api/v1/agentgroups/"+uid.String(),
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestAgentGroupController_Update_InvalidBody(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/api/v1/agentgroups/something",
		strings.NewReader("invalid"),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestAgentGroupController_Update_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	name := "g1"
	group := &agentgroupv1.AgentGroup{
		UID:        uuid.New(),
		Name:       name,
		Attributes: agentgroupv1.Attributes{},
		Selector: agentgroupv1.AgentSelector{
			IdentifyingAttributes:    map[string]string{},
			NonIdentifyingAttributes: map[string]string{},
		},
		CreatedAt: time.Now(),
		CreatedBy: "",
		DeletedAt: nil,
		DeletedBy: nil,
	}
	usecase.EXPECT().UpdateAgentGroup(mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)

	jsonBody, err := json.Marshal(group)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/api/v1/agentgroups/"+name,
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestAgentGroupController_Delete(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	uid := uuid.New()

	usecase.EXPECT().DeleteAgentGroup(mock.Anything, mock.Anything).Return(nil)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/agentgroups/"+uid.String(), nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestAgentGroupController_Delete_NotFound(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().DeleteAgentGroup(mock.Anything, mock.Anything).Return(port.ErrResourceNotExist)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/agentgroups/something", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestAgentGroupController_Delete_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentgroup.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().DeleteAgentGroup(mock.Anything, mock.Anything).Return(assert.AnError)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/agentgroups/something", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}
