package agentpackage_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agentpackage"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agentpackage/usecasemock"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const (
	testPackageName = "pkg1"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

func TestAgentPackageController_List(t *testing.T) {
	t.Parallel()

	t.Run("List AgentPackages - happycase", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := agentpackage.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		packages := []v1.AgentPackage{
			{
				Metadata: v1.AgentPackageMetadata{
					Name:       testPackageName,
					Attributes: v1.Attributes{},
				},
				Spec: v1.AgentPackageSpec{
					PackageType: "TopLevelPackageName",
					Version:     "1.0.0",
					DownloadURL: "https://example.com/pkg1.tar.gz",
				},
				Status: v1.AgentPackageStatus{
					Conditions: []v1.Condition{
						{
							Type:               v1.ConditionTypeCreated,
							LastTransitionTime: v1.NewTime(time.Now()),
							Status:             v1.ConditionStatusTrue,
							Reason:             "", Message: "Agent package created",
						},
					},
				},
			},
			{
				Metadata: v1.AgentPackageMetadata{
					Name:       "pkg2",
					Attributes: v1.Attributes{},
				},
				Spec: v1.AgentPackageSpec{
					PackageType: "AddonPackage",
					Version:     "2.0.0",
					DownloadURL: "https://example.com/pkg2.tar.gz",
				},
				Status: v1.AgentPackageStatus{
					Conditions: []v1.Condition{
						{
							Type:               v1.ConditionTypeCreated,
							LastTransitionTime: v1.NewTime(time.Now()),
							Status:             v1.ConditionStatusTrue,
							Reason:             "", Message: "Agent package created",
						},
					},
				},
			},
		}
		usecase.EXPECT().ListAgentPackages(mock.Anything, mock.Anything).Return(&v1.ListResponse[v1.AgentPackage]{
			Kind:       "AgentPackage",
			APIVersion: "v1",
			Metadata: v1.ListMeta{
				Continue:           "",
				RemainingItemCount: 0,
			},
			Items: packages,
		}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentpackages", nil)
		require.NoError(t, err)
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(2), gjson.Get(recorder.Body.String(), "items.#").Int())
	})

	t.Run("List AgentPackages - invalid limit", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := agentpackage.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentpackages?limit=invalid", nil)
		require.NoError(t, err)
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)

		// Check RFC 9457 structure
		body := recorder.Body.String()
		assert.Contains(t, body, "type")
		assert.Contains(t, body, "title")
		assert.Contains(t, body, "status")
		assert.Contains(t, body, "detail")
		assert.Contains(t, body, "instance")
		assert.Contains(t, body, "errors")

		// Check specific error information
		assert.Contains(t, body, "invalid format")
		assert.Contains(t, body, "query.limit")
		assert.Contains(t, body, "invalid")
	})

	t.Run("List AgentPackages - internal error", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := agentpackage.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		usecase.EXPECT().ListAgentPackages(mock.Anything, mock.Anything).Return(nil, assert.AnError)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentpackages", nil)
		require.NoError(t, err)
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestAgentPackageController_Get(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentpackage.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	agentPkg := &v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       testPackageName,
			Attributes: v1.Attributes{},
		},
		Spec: v1.AgentPackageSpec{
			PackageType: "TopLevelPackageName",
			Version:     "1.0.0",
			DownloadURL: "https://example.com/pkg1.tar.gz",
		},
		Status: v1.AgentPackageStatus{
			Conditions: []v1.Condition{
				{
					Type:               v1.ConditionTypeCreated,
					LastTransitionTime: v1.NewTime(time.Now()),
					Status:             v1.ConditionStatusTrue,
					Reason:             "", Message: "Agent package created",
				},
			},
		},
	}
	usecase.EXPECT().GetAgentPackage(mock.Anything, mock.Anything).Return(agentPkg, nil)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentpackages/pkg1", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestAgentPackageController_Get_NotFound(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentpackage.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().GetAgentPackage(mock.Anything, mock.Anything).Return(nil, port.ErrResourceNotExist)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentpackages/notfound", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestAgentPackageController_Get_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentpackage.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().GetAgentPackage(mock.Anything, mock.Anything).Return(nil, assert.AnError)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agentpackages/pkg1", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestAgentPackageController_Create(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentpackage.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	name := testPackageName
	returnValue := v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       name,
			Attributes: v1.Attributes{},
		},
		Spec: v1.AgentPackageSpec{
			PackageType: "TopLevelPackageName",
			Version:     "1.0.0",
			DownloadURL: "https://example.com/pkg1.tar.gz",
		},
		Status: v1.AgentPackageStatus{
			Conditions: []v1.Condition{
				{
					Type:               v1.ConditionTypeCreated,
					LastTransitionTime: v1.NewTime(time.Now()),
					Status:             v1.ConditionStatusTrue,
					Reason:             "", Message: "Agent package created",
				},
			},
		},
	}

	payload := v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       name,
			Attributes: v1.Attributes{},
		},
		Spec: v1.AgentPackageSpec{
			PackageType: "TopLevelPackageName",
			Version:     "1.0.0",
			DownloadURL: "https://example.com/pkg1.tar.gz",
		},
	}

	usecase.EXPECT().CreateAgentPackage(mock.Anything, mock.Anything).Return(&returnValue, nil)

	jsonBody, err := json.Marshal(payload)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/agentpackages",
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "/api/v1/agentpackages/"+name, recorder.Header().Get("Location"))
}

func TestAgentPackageController_Create_InvalidBody(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentpackage.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/agentpackages",
		strings.NewReader("invalid"),
	)
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	// Check RFC 9457 structure
	body := recorder.Body.String()
	assert.Contains(t, body, "type")
	assert.Contains(t, body, "title")
	assert.Contains(t, body, "status")
	assert.Contains(t, body, "detail")
	assert.Contains(t, body, "instance")
}

func TestAgentPackageController_Create_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentpackage.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	payload := v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       testPackageName,
			Attributes: v1.Attributes{},
		},
		Spec: v1.AgentPackageSpec{
			PackageType: "TopLevelPackageName",
			Version:     "1.0.0",
		},
	}

	usecase.EXPECT().CreateAgentPackage(mock.Anything, mock.Anything).Return(nil, assert.AnError)

	jsonBody, err := json.Marshal(payload)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/agentpackages",
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestAgentPackageController_Update(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentpackage.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	name := testPackageName
	pkg := &v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       name,
			Attributes: v1.Attributes{},
		},
		Spec: v1.AgentPackageSpec{
			PackageType: "TopLevelPackageName",
			Version:     "1.0.0",
			DownloadURL: "https://example.com/pkg1.tar.gz",
		},
		Status: v1.AgentPackageStatus{
			Conditions: []v1.Condition{
				{
					Type:               v1.ConditionTypeCreated,
					LastTransitionTime: v1.NewTime(time.Now()),
					Status:             v1.ConditionStatusTrue,
					Reason:             "", Message: "Agent package created",
				},
			},
		},
	}
	usecase.EXPECT().UpdateAgentPackage(mock.Anything, mock.Anything, mock.Anything).Return(pkg, nil)
	jsonBody, err := json.Marshal(pkg)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/api/v1/agentpackages/"+name,
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestAgentPackageController_Update_InvalidBody(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentpackage.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/api/v1/agentpackages/something",
		strings.NewReader("invalid"),
	)
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	// Check RFC 9457 structure
	body := recorder.Body.String()
	assert.Contains(t, body, "type")
	assert.Contains(t, body, "title")
	assert.Contains(t, body, "status")
	assert.Contains(t, body, "detail")
	assert.Contains(t, body, "instance")
}

func TestAgentPackageController_Update_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentpackage.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	name := testPackageName
	pkg := &v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       name,
			Attributes: v1.Attributes{},
		},
		Spec: v1.AgentPackageSpec{
			PackageType: "TopLevelPackageName",
			Version:     "1.0.0",
		},
		Status: v1.AgentPackageStatus{
			Conditions: []v1.Condition{
				{
					Type:               v1.ConditionTypeCreated,
					LastTransitionTime: v1.NewTime(time.Now()),
					Status:             v1.ConditionStatusTrue,
					Reason:             "", Message: "Agent package created",
				},
			},
		},
	}

	usecase.EXPECT().UpdateAgentPackage(mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)

	jsonBody, err := json.Marshal(pkg)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/api/v1/agentpackages/"+name,
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestAgentPackageController_Delete(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentpackage.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	name := testPackageName

	usecase.EXPECT().DeleteAgentPackage(mock.Anything, mock.Anything).Return(nil)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/agentpackages/"+name, nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestAgentPackageController_Delete_NotFound(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentpackage.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().DeleteAgentPackage(mock.Anything, mock.Anything).Return(port.ErrResourceNotExist)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/agentpackages/something", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestAgentPackageController_Delete_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := agentpackage.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().DeleteAgentPackage(mock.Anything, mock.Anything).Return(assert.AnError)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/agentpackages/something", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}
