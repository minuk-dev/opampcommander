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
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agentpackage"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agentpackage/usecasemock"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	agentpackagesvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agentpackage"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const (
	testPackageName = "pkg1"
	testNamespace   = "default"
	testBaseURL     = "/api/v1/namespaces/default/agentpackages"
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
				Metadata: v1.AgentPackageMetadata{ResourceVersion: 1,
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
				Metadata: v1.AgentPackageMetadata{ResourceVersion: 1,
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
		usecase.EXPECT().ListAgentPackages(mock.Anything, "default",
			mock.Anything).Return(&v1.ListResponse[v1.AgentPackage]{
			Kind:       "AgentPackage",
			APIVersion: "v1",
			Metadata: v1.ListMeta{
				Continue:           "",
				RemainingItemCount: 0,
			},
			Items: packages,
		}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, testBaseURL, nil)
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
		listURL := testBaseURL + "?limit=invalid"
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, listURL, nil)
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

		usecase.EXPECT().ListAgentPackages(mock.Anything, "default", mock.Anything).Return(nil, assert.AnError)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, testBaseURL, nil)
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
		Metadata: v1.AgentPackageMetadata{ResourceVersion: 1,
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
	usecase.EXPECT().GetAgentPackage(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(agentPkg, nil)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, testBaseURL+"/pkg1", nil)
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

	usecase.EXPECT().GetAgentPackage(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, model.ErrResourceNotExist)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, testBaseURL+"/notfound", nil)
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

	usecase.EXPECT().GetAgentPackage(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, testBaseURL+"/pkg1", nil)
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
		Metadata: v1.AgentPackageMetadata{ResourceVersion: 1,
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
		Metadata: v1.AgentPackageMetadata{ResourceVersion: 1,
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
		testBaseURL,
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, testBaseURL+"/"+name, recorder.Header().Get("Location"))
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
		testBaseURL,
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
		Metadata: v1.AgentPackageMetadata{ResourceVersion: 1,
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
		testBaseURL,
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
		Metadata: v1.AgentPackageMetadata{ResourceVersion: 1,
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
	usecase.EXPECT().UpdateAgentPackage(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pkg, nil)
	jsonBody, err := json.Marshal(pkg)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		testBaseURL+"/"+name,
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
		testBaseURL+"/something",
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
		Metadata: v1.AgentPackageMetadata{ResourceVersion: 1,
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

	usecase.EXPECT().UpdateAgentPackage(
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil, assert.AnError)

	jsonBody, err := json.Marshal(pkg)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		testBaseURL+"/"+name,
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

	usecase.EXPECT().DeleteAgentPackage(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete,
		testBaseURL+"/"+name+"?resourceVersion=1", nil)
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

	usecase.EXPECT().DeleteAgentPackage(mock.Anything,
		mock.Anything, mock.Anything, mock.Anything).Return(model.ErrResourceNotExist)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete,
		testBaseURL+"/something?resourceVersion=1", nil)
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

	usecase.EXPECT().DeleteAgentPackage(mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(assert.AnError)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete,
		testBaseURL+"/something?resourceVersion=1", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestAgentPackageController_ConditionalMutations(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	ctrlBase := testutil.NewBase(t).ForController()
	repo := inmemory.NewAgentPackageRepository()
	domain := agentservice.NewAgentPackageService(repo)
	service := agentpackagesvc.NewAgentPackageService(domain, ctrlBase.Logger)
	ctrlBase.SetupRouter(agentpackage.NewController(service, ctrlBase.Logger))
	server := httptest.NewServer(ctrlBase.Router)
	t.Cleanup(server.Close)
	cli := client.New(server.URL).AgentPackageService
	resource := &v1.AgentPackage{Metadata: v1.AgentPackageMetadata{Name: "package",
		Namespace: "default"}, Spec: v1.AgentPackageSpec{Version: "v1"}}
	created, err := cli.CreateAgentPackage(ctx, "default", resource)
	require.NoError(t, err)
	require.EqualValues(t, 1, created.Metadata.ResourceVersion)
	// A lost create response can be retried without overwriting the winner.
	_, err = cli.CreateAgentPackage(ctx, "default", resource)
	assertHTTPStatus(t, err, http.StatusConflict)
	stale, err := cli.GetAgentPackage(ctx, "default", "package")
	require.NoError(t, err)

	created.Spec.Version = "v2"
	updated, err := cli.UpdateAgentPackage(ctx, created)
	require.NoError(t, err)
	require.EqualValues(t, 2, updated.Metadata.ResourceVersion)
	// A lost update response retried with the same desired state is a no-op.
	retried, err := cli.UpdateAgentPackage(ctx, created)
	require.NoError(t, err)
	require.Equal(t, updated.Metadata.ResourceVersion, retried.Metadata.ResourceVersion)
	unchanged, err := cli.UpdateAgentPackage(ctx, updated)
	require.NoError(t, err)
	require.Equal(t, updated.Metadata.ResourceVersion, unchanged.Metadata.ResourceVersion)

	stale.Spec.Version = "v3"
	_, err = cli.UpdateAgentPackage(ctx, stale)
	assertHTTPStatus(t, err, http.StatusConflict)
	err = cli.DeleteAgentPackage(ctx, "default", "package", stale.Metadata.ResourceVersion)
	assertHTTPStatus(t, err, http.StatusConflict)

	missing := *updated
	missing.Metadata.ResourceVersion = 0
	_, err = cli.UpdateAgentPackage(ctx, &missing)
	assertHTTPStatus(t, err, http.StatusBadRequest)
	err = cli.DeleteAgentPackage(ctx, "default", "package", 0)
	assertHTTPStatus(t, err, http.StatusBadRequest)
	// Path/body mismatch is rejected without changing either record.
	body, err := json.Marshal(updated)
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPut, server.URL+testBaseURL+"/other", strings.NewReader(string(body)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	// Identity validation happens before existence lookup at the API boundary.
	recorder := httptest.NewRecorder()
	ctrlBase.Router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.NoError(t, cli.DeleteAgentPackage(ctx, "default", "package", updated.Metadata.ResourceVersion))
	// A lost delete response retried against its tombstone succeeds without another write.
	require.NoError(t, cli.DeleteAgentPackage(ctx, "default", "package", updated.Metadata.ResourceVersion))
	tombstone, err := cli.GetAgentPackage(ctx, "default", "package", client.WithGetIncludeDeleted(true))
	require.NoError(t, err)
	require.EqualValues(t, 3, tombstone.Metadata.ResourceVersion)

	_, err = cli.CreateAgentPackage(ctx, "default", resource)
	assertHTTPStatus(t, err, http.StatusConflict)
}

func assertHTTPStatus(t *testing.T, err error, status int) {
	t.Helper()

	var responseErr *client.ResponseError
	require.ErrorAs(t, err, &responseErr)
	require.Equal(t, status, responseErr.StatusCode)
}

func TestAgentPackageController_MergePatch(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	base := testutil.NewBase(t).ForController()
	domain := agentservice.NewAgentPackageService(inmemory.NewAgentPackageRepository())
	svc := agentpackagesvc.NewAgentPackageService(domain, base.Logger)
	base.SetupRouter(agentpackage.NewController(svc, base.Logger))

	server := httptest.NewServer(base.Router)
	defer server.Close()

	cli := client.New(server.URL).AgentPackageService
	_, err := cli.CreateAgentPackage(ctx, "default", &v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{Name: "pkg", Namespace: "default"},
		Spec:     v1.AgentPackageSpec{Version: "v1", DownloadURL: "https://example.com/package"},
	})
	require.NoError(t, err)

	body := []byte(`{"metadata":{"resourceVersion":"1"},"spec":{"version":"v2"}}`)
	updated, err := cli.PatchAgentPackage(ctx, "default", "pkg", body)
	require.NoError(t, err)
	require.Equal(t, "v2", updated.Spec.Version)
	require.Equal(t, "https://example.com/package", updated.Spec.DownloadURL)
	require.EqualValues(t, 2, updated.Metadata.ResourceVersion)
	updated, err = cli.PatchAgentPackage(ctx, "default", "pkg", body)
	require.NoError(t, err)
	require.EqualValues(t, 2, updated.Metadata.ResourceVersion)

	_, err = cli.PatchAgentPackage(ctx, "default", "pkg",
		[]byte(`{"metadata":{"resourceVersion":"1"},"spec":{"version":"v3"}}`))
	assertHTTPStatus(t, err, http.StatusConflict)

	for _, body := range []string{`null`, `{"status":{}}`, `{"metadata":{"resourceVersion":null}}`,
		`{"spec":{"unknown":true}}`} {
		_, err = cli.PatchAgentPackage(ctx, "default", "pkg", []byte(body))
		assertHTTPStatus(t, err, http.StatusBadRequest)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/api/v1/namespaces/default/agentpackages/pkg",
		strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	base.Router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusUnsupportedMediaType, recorder.Code)
	require.Equal(t, "application/merge-patch+json", recorder.Header().Get("Accept-Patch"))
	require.NoError(t, cli.DeleteAgentPackage(ctx, "default", "pkg", 2))
	_, err = cli.PatchAgentPackage(ctx, "default", "pkg", []byte(`{}`))
	assertHTTPStatus(t, err, http.StatusNotFound)
	_, err = cli.PatchAgentPackage(ctx, "default", "missing", []byte(`{}`))
	assertHTTPStatus(t, err, http.StatusNotFound)
}
