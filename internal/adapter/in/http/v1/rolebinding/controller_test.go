package rolebinding_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/goleak"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/rolebinding"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/rolebinding/usecasemock"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

func TestRoleBindingController_List(t *testing.T) {
	t.Parallel()

	t.Run("List RoleBindings - happycase", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := rolebinding.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		bindings := []v1.RoleBinding{
			{
				Kind:       v1.RoleBindingKind,
				APIVersion: v1.APIVersion,
				Metadata: v1.RoleBindingMetadata{
					Namespace: "production",
					Name:      "viewer-binding",
				},
				Spec: v1.RoleBindingSpec{
					RoleRef:  v1.RoleBindingRoleRef{Kind: "Role", Name: "Viewer"},
					Subjects: []v1.RoleBindingSubject{{Kind: "User", Name: "alice@example.com"}},
				},
			},
			{
				Kind:       v1.RoleBindingKind,
				APIVersion: v1.APIVersion,
				Metadata: v1.RoleBindingMetadata{
					Namespace: "production",
					Name:      "admin-binding",
				},
				Spec: v1.RoleBindingSpec{
					RoleRef:  v1.RoleBindingRoleRef{Kind: "Role", Name: "Admin"},
					Subjects: []v1.RoleBindingSubject{{Kind: "User", Name: "bob@example.com"}},
				},
			},
		}
		usecase.EXPECT().ListRoleBindings(mock.Anything, mock.Anything).Return(&v1.ListResponse[v1.RoleBinding]{
			Kind:       v1.RoleBindingKind,
			APIVersion: v1.APIVersion,
			Metadata: v1.ListMeta{
				Continue:           "",
				RemainingItemCount: 0,
			},
			Items: bindings,
		}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/production/rolebindings", nil,
		)
		require.NoError(t, err)
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(2), gjson.Get(recorder.Body.String(), "items.#").Int())
	})

	t.Run("List RoleBindings - invalid limit", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := rolebinding.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/production/rolebindings?limit=invalid", nil,
		)
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

	t.Run("List RoleBindings - internal error", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := rolebinding.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		usecase.EXPECT().ListRoleBindings(mock.Anything, mock.Anything).Return(nil, assert.AnError)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet,
			"/api/v1/namespaces/production/rolebindings", nil,
		)
		require.NoError(t, err)
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestRoleBindingController_Get(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := rolebinding.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	rb := &v1.RoleBinding{
		Kind:       v1.RoleBindingKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.RoleBindingMetadata{
			Namespace: "production",
			Name:      "viewer-binding",
		},
		Spec: v1.RoleBindingSpec{
			RoleRef:  v1.RoleBindingRoleRef{Kind: "Role", Name: "Viewer"},
			Subjects: []v1.RoleBindingSubject{{Kind: "User", Name: "alice@example.com"}},
		},
		Status: v1.RoleBindingStatus{},
	}
	usecase.EXPECT().GetRoleBinding(mock.Anything, mock.Anything, mock.Anything).Return(rb, nil)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet,
		"/api/v1/namespaces/production/rolebindings/viewer-binding", nil,
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestRoleBindingController_Get_NotFound(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := rolebinding.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().
		GetRoleBinding(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, port.ErrResourceNotExist)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet,
		"/api/v1/namespaces/production/rolebindings/notfound", nil,
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestRoleBindingController_Get_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := rolebinding.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().
		GetRoleBinding(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet,
		"/api/v1/namespaces/production/rolebindings/viewer-binding", nil,
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestRoleBindingController_Create(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := rolebinding.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	returnValue := v1.RoleBinding{
		Kind:       v1.RoleBindingKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.RoleBindingMetadata{
			Namespace: "production",
			Name:      "viewer-binding",
		},
		Spec: v1.RoleBindingSpec{
			RoleRef:  v1.RoleBindingRoleRef{Kind: "Role", Name: "Viewer"},
			Subjects: []v1.RoleBindingSubject{{Kind: "User", Name: "alice@example.com"}},
		},
		Status: v1.RoleBindingStatus{},
	}

	payload := v1.RoleBinding{
		Kind:       v1.RoleBindingKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.RoleBindingMetadata{
			Name: "viewer-binding",
		},
		Spec: v1.RoleBindingSpec{
			RoleRef:  v1.RoleBindingRoleRef{Kind: "Role", Name: "Viewer"},
			Subjects: []v1.RoleBindingSubject{{Kind: "User", Name: "alice@example.com"}},
		},
	}

	usecase.EXPECT().CreateRoleBinding(mock.Anything, mock.Anything).Return(&returnValue, nil)

	jsonBody, err := json.Marshal(payload)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/namespaces/production/rolebindings",
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "/api/v1/namespaces/production/rolebindings/viewer-binding", recorder.Header().Get("Location"))
}

func TestRoleBindingController_Create_InvalidBody(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := rolebinding.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/namespaces/production/rolebindings",
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

func TestRoleBindingController_Create_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := rolebinding.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	payload := v1.RoleBinding{
		Kind:       v1.RoleBindingKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.RoleBindingMetadata{
			Name: "viewer-binding",
		},
		Spec: v1.RoleBindingSpec{
			RoleRef:  v1.RoleBindingRoleRef{Kind: "Role", Name: "Viewer"},
			Subjects: []v1.RoleBindingSubject{{Kind: "User", Name: "alice@example.com"}},
		},
	}

	usecase.EXPECT().CreateRoleBinding(mock.Anything, mock.Anything).Return(nil, assert.AnError)

	jsonBody, err := json.Marshal(payload)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/namespaces/production/rolebindings",
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestRoleBindingController_Update(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := rolebinding.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	rb := &v1.RoleBinding{
		Kind:       v1.RoleBindingKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.RoleBindingMetadata{
			Namespace: "production",
			Name:      "viewer-binding",
		},
		Spec: v1.RoleBindingSpec{
			RoleRef:  v1.RoleBindingRoleRef{Kind: "Role", Name: "Viewer"},
			Subjects: []v1.RoleBindingSubject{{Kind: "User", Name: "alice@example.com"}},
		},
		Status: v1.RoleBindingStatus{},
	}
	usecase.EXPECT().
		UpdateRoleBinding(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(rb, nil)
	jsonBody, err := json.Marshal(rb)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/api/v1/namespaces/production/rolebindings/viewer-binding",
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestRoleBindingController_Update_InvalidBody(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := rolebinding.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/api/v1/namespaces/production/rolebindings/viewer-binding",
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

func TestRoleBindingController_Update_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := rolebinding.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	rb := &v1.RoleBinding{
		Kind:       v1.RoleBindingKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.RoleBindingMetadata{
			Namespace: "production",
			Name:      "viewer-binding",
		},
		Spec: v1.RoleBindingSpec{
			RoleRef:  v1.RoleBindingRoleRef{Kind: "Role", Name: "Viewer"},
			Subjects: []v1.RoleBindingSubject{{Kind: "User", Name: "alice@example.com"}},
		},
	}

	usecase.EXPECT().
		UpdateRoleBinding(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	jsonBody, err := json.Marshal(rb)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/api/v1/namespaces/production/rolebindings/viewer-binding",
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestRoleBindingController_Delete(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := rolebinding.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().DeleteRoleBinding(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodDelete,
		"/api/v1/namespaces/production/rolebindings/viewer-binding", nil,
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestRoleBindingController_Delete_NotFound(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := rolebinding.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().
		DeleteRoleBinding(mock.Anything, mock.Anything, mock.Anything).
		Return(port.ErrResourceNotExist)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodDelete,
		"/api/v1/namespaces/production/rolebindings/notfound", nil,
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestRoleBindingController_Delete_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := rolebinding.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().
		DeleteRoleBinding(mock.Anything, mock.Anything, mock.Anything).
		Return(assert.AnError)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodDelete,
		"/api/v1/namespaces/production/rolebindings/something", nil,
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}
