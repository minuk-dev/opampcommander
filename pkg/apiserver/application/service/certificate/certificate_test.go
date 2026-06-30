package certificate_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	certificatesvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/certificate"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

// mockCertificateUsecase is a mock implementation of agentport.CertificateUsecase.
type mockCertificateUsecase struct {
	mock.Mock
}

func (m *mockCertificateUsecase) GetCertificate(
	ctx context.Context, namespace, name string, options *model.GetOptions,
) (*agentmodel.Certificate, error) {
	args := m.Called(ctx, namespace, name, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cert, ok := args.Get(0).(*agentmodel.Certificate)
	if !ok {
		return nil, errMock
	}

	return cert, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockCertificateUsecase) SaveCertificate(
	ctx context.Context, certificate *agentmodel.Certificate,
) (*agentmodel.Certificate, error) {
	args := m.Called(ctx, certificate)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cert, ok := args.Get(0).(*agentmodel.Certificate)
	if !ok {
		return nil, errMock
	}

	return cert, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockCertificateUsecase) CreateCertificate(
	ctx context.Context, certificate *agentmodel.Certificate, actor string,
) (*agentmodel.Certificate, error) {
	args := m.Called(ctx, certificate, actor)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cert, ok := args.Get(0).(*agentmodel.Certificate)
	if !ok {
		return nil, errMock
	}

	return cert, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockCertificateUsecase) UpdateCertificate(
	ctx context.Context, namespace, name string, certificate *agentmodel.Certificate, actor string,
) (*agentmodel.Certificate, error) {
	args := m.Called(ctx, namespace, name, certificate, actor)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cert, ok := args.Get(0).(*agentmodel.Certificate)
	if !ok {
		return nil, errMock
	}

	return cert, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockCertificateUsecase) ListCertificate(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Certificate], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Certificate])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockCertificateUsecase) DeleteCertificate(
	ctx context.Context, namespace, name string, deletedAt time.Time, deletedBy string,
) (*agentmodel.Certificate, error) {
	args := m.Called(ctx, namespace, name, deletedAt, deletedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cert, ok := args.Get(0).(*agentmodel.Certificate)
	if !ok {
		return nil, errMock
	}

	return cert, args.Error(1) //nolint:wrapcheck // mock error
}

func newSvc(t *testing.T, cert *mockCertificateUsecase) *certificatesvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return certificatesvc.NewCertificateService(cert, base.Logger)
}

func newCert(namespace, name string) *agentmodel.Certificate {
	return &agentmodel.Certificate{
		Metadata: agentmodel.CertificateMetadata{Namespace: namespace, Name: name},
		Spec:     agentmodel.CertificateSpec{Cert: []byte("cert"), PrivateKey: []byte("key")},
	}
}

func apiCert(namespace, name string) *v1.Certificate {
	return &v1.Certificate{
		Kind:       v1.CertificateKind,
		APIVersion: v1.APIVersion,
		Metadata:   v1.CertificateMetadata{Namespace: namespace, Name: name},
	}
}

func TestService_GetCertificate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockCert := new(mockCertificateUsecase)
		svc := newSvc(t, mockCert)

		mockCert.On("GetCertificate", ctx, "default", "cert-1", (*model.GetOptions)(nil)).
			Return(newCert("default", "cert-1"), nil)

		result, err := svc.GetCertificate(ctx, "default", "cert-1", nil)

		require.NoError(t, err)
		assert.Equal(t, "default", result.Metadata.Namespace)
		assert.Equal(t, "cert-1", result.Metadata.Name)
		mockCert.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockCert := new(mockCertificateUsecase)
		svc := newSvc(t, mockCert)

		mockCert.On("GetCertificate", ctx, "default", "missing", (*model.GetOptions)(nil)).Return(nil, errMock)

		result, err := svc.GetCertificate(ctx, "default", "missing", nil)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "get certificate")
		mockCert.AssertExpectations(t)
	})
}

func TestService_ListCertificates(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockCert := new(mockCertificateUsecase)
		svc := newSvc(t, mockCert)

		opts := &applicationport.ListOptions{Limit: 10}
		resp := &model.ListResponse[*agentmodel.Certificate]{
			Items:    []*agentmodel.Certificate{newCert("default", "cert-1")},
			Continue: "next",
		}
		mockCert.On("ListCertificate", ctx, opts.ToDomain()).Return(resp, nil)

		result, err := svc.ListCertificates(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, v1.CertificateKind, result.Kind)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, "next", result.Metadata.Continue)
		mockCert.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockCert := new(mockCertificateUsecase)
		svc := newSvc(t, mockCert)

		opts := &applicationport.ListOptions{Limit: 10}
		mockCert.On("ListCertificate", ctx, opts.ToDomain()).Return(nil, errMock)

		result, err := svc.ListCertificates(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "list certificates")
		mockCert.AssertExpectations(t)
	})
}

func TestService_CreateCertificate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockCert := new(mockCertificateUsecase)
		svc := newSvc(t, mockCert)

		mockCert.On("CreateCertificate", ctx, mock.MatchedBy(func(c *agentmodel.Certificate) bool {
			return c.Metadata.Name == "cert-1"
		}), mock.AnythingOfType("string")).Return(newCert("default", "cert-1"), nil)

		result, err := svc.CreateCertificate(ctx, apiCert("default", "cert-1"))

		require.NoError(t, err)
		assert.Equal(t, "cert-1", result.Metadata.Name)
		mockCert.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockCert := new(mockCertificateUsecase)
		svc := newSvc(t, mockCert)

		mockCert.On("CreateCertificate", ctx, mock.Anything, mock.AnythingOfType("string")).Return(nil, errMock)

		result, err := svc.CreateCertificate(ctx, apiCert("default", "cert-1"))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "create certificate")
		mockCert.AssertExpectations(t)
	})
}

func TestService_UpdateCertificate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockCert := new(mockCertificateUsecase)
		svc := newSvc(t, mockCert)

		mockCert.On("UpdateCertificate", ctx, "default", "cert-1", mock.Anything, mock.AnythingOfType("string")).
			Return(newCert("default", "cert-1"), nil)

		result, err := svc.UpdateCertificate(ctx, "default", "cert-1", apiCert("default", "cert-1"))

		require.NoError(t, err)
		assert.Equal(t, "cert-1", result.Metadata.Name)
		mockCert.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockCert := new(mockCertificateUsecase)
		svc := newSvc(t, mockCert)

		mockCert.On("UpdateCertificate", ctx, "default", "cert-1", mock.Anything, mock.AnythingOfType("string")).
			Return(nil, errMock)

		result, err := svc.UpdateCertificate(ctx, "default", "cert-1", apiCert("default", "cert-1"))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "update certificate")
		mockCert.AssertExpectations(t)
	})
}

func TestService_DeleteCertificate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockCert := new(mockCertificateUsecase)
		svc := newSvc(t, mockCert)

		mockCert.On("DeleteCertificate", ctx, "default", "cert-1",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("string")).
			Return(newCert("default", "cert-1"), nil)

		err := svc.DeleteCertificate(ctx, "default", "cert-1")

		require.NoError(t, err)
		mockCert.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockCert := new(mockCertificateUsecase)
		svc := newSvc(t, mockCert)

		mockCert.On("DeleteCertificate", ctx, "default", "cert-1",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("string")).
			Return(nil, errMock)

		err := svc.DeleteCertificate(ctx, "default", "cert-1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete certificate")
		mockCert.AssertExpectations(t)
	})
}
