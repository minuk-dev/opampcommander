package certificate_test

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
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/certificate"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/certificate/usecasemock"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const (
	testCertName = "test-cert"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

func TestCertificateController_List(t *testing.T) {
	t.Parallel()

	t.Run("List Certificates - happycase", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := certificate.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		certificates := []v1.Certificate{
			{
				Metadata: v1.CertificateMetadata{
					Name:       testCertName,
					Attributes: v1.Attributes{},
				},
				Spec: v1.CertificateSpec{
					Cert: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
				},
				Status: v1.CertificateStatus{
					Conditions: []v1.Condition{
						{
							Type:               v1.ConditionTypeCreated,
							LastTransitionTime: v1.NewTime(time.Now()),
							Status:             v1.ConditionStatusTrue,
							Reason:             "", Message: "Certificate created",
						},
					},
				},
			},
			{
				Metadata: v1.CertificateMetadata{
					Name:       "cert2",
					Attributes: v1.Attributes{},
				},
				Spec: v1.CertificateSpec{
					Cert: "-----BEGIN CERTIFICATE-----\ntest2\n-----END CERTIFICATE-----",
				},
				Status: v1.CertificateStatus{
					Conditions: []v1.Condition{
						{
							Type:               v1.ConditionTypeCreated,
							LastTransitionTime: v1.NewTime(time.Now()),
							Status:             v1.ConditionStatusTrue,
							Reason:             "", Message: "Certificate created",
						},
					},
				},
			},
		}
		usecase.EXPECT().ListCertificates(mock.Anything, mock.Anything).Return(&v1.ListResponse[v1.Certificate]{
			Kind:       "Certificate",
			APIVersion: "v1",
			Metadata: v1.ListMeta{
				Continue:           "",
				RemainingItemCount: 0,
			},
			Items: certificates,
		}, nil)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/certificates", nil)
		require.NoError(t, err)
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, int64(2), gjson.Get(recorder.Body.String(), "items.#").Int())
	})

	t.Run("List Certificates - invalid limit", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := certificate.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router
		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/certificates?limit=invalid", nil)
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

	t.Run("List Certificates - internal error", func(t *testing.T) {
		t.Parallel()
		ctrlBase := testutil.NewBase(t).ForController()
		usecase := usecasemock.NewMockUsecase(t)
		controller := certificate.NewController(usecase, ctrlBase.Logger)
		ctrlBase.SetupRouter(controller)
		router := ctrlBase.Router

		usecase.EXPECT().ListCertificates(mock.Anything, mock.Anything).Return(nil, assert.AnError)

		recorder := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/certificates", nil)
		require.NoError(t, err)
		router.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func TestCertificateController_Get(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := certificate.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	cert := &v1.Certificate{
		Metadata: v1.CertificateMetadata{
			Name:       testCertName,
			Attributes: v1.Attributes{},
		},
		Spec: v1.CertificateSpec{
			Cert: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		},
		Status: v1.CertificateStatus{
			Conditions: []v1.Condition{
				{
					Type:               v1.ConditionTypeCreated,
					LastTransitionTime: v1.NewTime(time.Now()),
					Status:             v1.ConditionStatusTrue,
					Reason:             "", Message: "Certificate created",
				},
			},
		},
	}
	usecase.EXPECT().GetCertificate(mock.Anything, mock.Anything).Return(cert, nil)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/certificates/"+testCertName, nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestCertificateController_Get_NotFound(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := certificate.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().GetCertificate(mock.Anything, mock.Anything).Return(nil, port.ErrResourceNotExist)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/certificates/notfound", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestCertificateController_Get_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := certificate.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().GetCertificate(mock.Anything, mock.Anything).Return(nil, assert.AnError)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/certificates/"+testCertName, nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestCertificateController_Create(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := certificate.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	name := testCertName
	returnValue := v1.Certificate{
		Metadata: v1.CertificateMetadata{
			Name:       name,
			Attributes: v1.Attributes{},
		},
		Spec: v1.CertificateSpec{
			Cert: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		},
		Status: v1.CertificateStatus{
			Conditions: []v1.Condition{
				{
					Type:               v1.ConditionTypeCreated,
					LastTransitionTime: v1.NewTime(time.Now()),
					Status:             v1.ConditionStatusTrue,
					Reason:             "", Message: "Certificate created",
				},
			},
		},
	}

	payload := v1.Certificate{
		Metadata: v1.CertificateMetadata{
			Name:       name,
			Attributes: v1.Attributes{},
		},
		Spec: v1.CertificateSpec{
			Cert: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		},
	}

	usecase.EXPECT().CreateCertificate(mock.Anything, mock.Anything).Return(&returnValue, nil)

	jsonBody, err := json.Marshal(payload)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/certificates",
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "/api/v1/certificates/"+name, recorder.Header().Get("Location"))
}

func TestCertificateController_Create_InvalidBody(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := certificate.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/certificates",
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

func TestCertificateController_Create_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := certificate.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	payload := v1.Certificate{
		Metadata: v1.CertificateMetadata{
			Name:       testCertName,
			Attributes: v1.Attributes{},
		},
		Spec: v1.CertificateSpec{
			Cert: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		},
	}

	usecase.EXPECT().CreateCertificate(mock.Anything, mock.Anything).Return(nil, assert.AnError)

	jsonBody, err := json.Marshal(payload)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/certificates",
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestCertificateController_Update(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := certificate.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	name := testCertName
	cert := &v1.Certificate{
		Metadata: v1.CertificateMetadata{
			Name:       name,
			Attributes: v1.Attributes{},
		},
		Spec: v1.CertificateSpec{
			Cert: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		},
		Status: v1.CertificateStatus{
			Conditions: []v1.Condition{
				{
					Type:               v1.ConditionTypeCreated,
					LastTransitionTime: v1.NewTime(time.Now()),
					Status:             v1.ConditionStatusTrue,
					Reason:             "", Message: "Certificate created",
				},
			},
		},
	}
	usecase.EXPECT().UpdateCertificate(mock.Anything, mock.Anything, mock.Anything).Return(cert, nil)
	jsonBody, err := json.Marshal(cert)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/api/v1/certificates/"+name,
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestCertificateController_Update_InvalidBody(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := certificate.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/api/v1/certificates/something",
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

func TestCertificateController_Update_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := certificate.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	name := testCertName
	cert := &v1.Certificate{
		Metadata: v1.CertificateMetadata{
			Name:       name,
			Attributes: v1.Attributes{},
		},
		Spec: v1.CertificateSpec{
			Cert: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		},
		Status: v1.CertificateStatus{
			Conditions: []v1.Condition{
				{
					Type:               v1.ConditionTypeCreated,
					LastTransitionTime: v1.NewTime(time.Now()),
					Status:             v1.ConditionStatusTrue,
					Reason:             "", Message: "Certificate created",
				},
			},
		},
	}

	usecase.EXPECT().UpdateCertificate(mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)

	jsonBody, err := json.Marshal(cert)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/api/v1/certificates/"+name,
		strings.NewReader(string(jsonBody)),
	)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestCertificateController_Delete(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := certificate.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router
	name := testCertName

	usecase.EXPECT().DeleteCertificate(mock.Anything, mock.Anything).Return(nil)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/certificates/"+name, nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestCertificateController_Delete_NotFound(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := certificate.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().DeleteCertificate(mock.Anything, mock.Anything).Return(port.ErrResourceNotExist)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/certificates/something", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestCertificateController_Delete_InternalError(t *testing.T) {
	t.Parallel()
	ctrlBase := testutil.NewBase(t).ForController()
	usecase := usecasemock.NewMockUsecase(t)
	controller := certificate.NewController(usecase, ctrlBase.Logger)
	ctrlBase.SetupRouter(controller)
	router := ctrlBase.Router

	usecase.EXPECT().DeleteCertificate(mock.Anything, mock.Anything).Return(assert.AnError)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/certificates/something", nil)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}
