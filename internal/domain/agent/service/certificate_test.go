package agentservice_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/internal/domain/agent/service"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var (
	errCertificatePersistence = errors.New("certificate persistence error")
)

// MockCertificatePersistencePort is a mock implementation of CertificatePersistencePort.
type MockCertificatePersistencePort struct {
	mock.Mock
}

func (m *MockCertificatePersistencePort) GetCertificate(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.Certificate, error) {
	args := m.Called(ctx, namespace, name, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cert, ok := args.Get(0).(*agentmodel.Certificate)
	if !ok {
		return nil, errUnexpectedType
	}

	return cert, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockCertificatePersistencePort) PutCertificate(
	ctx context.Context,
	certificate *agentmodel.Certificate,
) (*agentmodel.Certificate, error) {
	args := m.Called(ctx, certificate)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cert, ok := args.Get(0).(*agentmodel.Certificate)
	if !ok {
		return nil, errUnexpectedType
	}

	return cert, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockCertificatePersistencePort) ListCertificate(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Certificate], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Certificate])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

// Ensure MockCertificatePersistencePort implements the interface.
var _ agentport.CertificatePersistencePort = (*MockCertificatePersistencePort)(nil)

func TestCertificateService_GetCertificate(t *testing.T) {
	t.Parallel()

	t.Run("Successfully get certificate", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPort := new(MockCertificatePersistencePort)
		logger := slog.Default()

		certService := agentservice.NewCertificateService(mockPort, logger)

		expectedCert := &agentmodel.Certificate{
			Metadata: agentmodel.CertificateMetadata{
				Name:       "test-cert",
				Attributes: agentmodel.Attributes{"env": "prod"},
			},
			Spec: agentmodel.CertificateSpec{
				Cert:       []byte("test-cert-data"),
				PrivateKey: []byte("test-key-data"),
				CaCert:     []byte("test-ca-data"),
			},
			Status: agentmodel.CertificateStatus{},
		}

		mockPort.On("GetCertificate", ctx, "default", "test-cert", (*model.GetOptions)(nil)).Return(expectedCert, nil)

		cert, err := certService.GetCertificate(ctx, "default", "test-cert", nil)

		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.Equal(t, "test-cert", cert.Metadata.Name)
		assert.Equal(t, []byte("test-cert-data"), cert.Spec.Cert)
		mockPort.AssertExpectations(t)
	})

	t.Run("Certificate not found", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPort := new(MockCertificatePersistencePort)
		logger := slog.Default()

		certService := agentservice.NewCertificateService(mockPort, logger)

		mockPort.On("GetCertificate", ctx, "default", "non-existent", (*model.GetOptions)(nil)).
			Return(nil, port.ErrResourceNotExist)

		cert, err := certService.GetCertificate(ctx, "default", "non-existent", nil)

		require.Error(t, err)
		assert.Nil(t, cert)
		assert.Contains(t, err.Error(), "failed to get certificate from persistence")
		mockPort.AssertExpectations(t)
	})

	t.Run("Persistence error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPort := new(MockCertificatePersistencePort)
		logger := slog.Default()

		certService := agentservice.NewCertificateService(mockPort, logger)

		mockPort.On("GetCertificate", ctx, "default", "test-cert", (*model.GetOptions)(nil)).
			Return(nil, errCertificatePersistence)

		cert, err := certService.GetCertificate(ctx, "default", "test-cert", nil)

		require.Error(t, err)
		assert.Nil(t, cert)
		mockPort.AssertExpectations(t)
	})
}

func TestCertificateService_ListCertificate(t *testing.T) {
	t.Parallel()

	t.Run("Successfully list certificates", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPort := new(MockCertificatePersistencePort)
		logger := slog.Default()

		certService := agentservice.NewCertificateService(mockPort, logger)

		certs := []*agentmodel.Certificate{
			{
				Metadata: agentmodel.CertificateMetadata{Name: "cert-1"},
				Spec:     agentmodel.CertificateSpec{Cert: []byte("cert-1-data")},
				Status:   agentmodel.CertificateStatus{},
			},
			{
				Metadata: agentmodel.CertificateMetadata{Name: "cert-2"},
				Spec:     agentmodel.CertificateSpec{Cert: []byte("cert-2-data")},
				Status:   agentmodel.CertificateStatus{},
			},
		}

		expectedResp := &model.ListResponse[*agentmodel.Certificate]{
			Items:              certs,
			Continue:           "",
			RemainingItemCount: 0,
		}

		options := &model.ListOptions{Limit: 10}
		mockPort.On("ListCertificate", ctx, options).Return(expectedResp, nil)

		resp, err := certService.ListCertificate(ctx, options)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Items, 2)
		assert.Equal(t, "cert-1", resp.Items[0].Metadata.Name)
		assert.Equal(t, "cert-2", resp.Items[1].Metadata.Name)
		mockPort.AssertExpectations(t)
	})

	t.Run("Empty list", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPort := new(MockCertificatePersistencePort)
		logger := slog.Default()

		certService := agentservice.NewCertificateService(mockPort, logger)

		expectedResp := &model.ListResponse[*agentmodel.Certificate]{
			Items:              []*agentmodel.Certificate{},
			Continue:           "",
			RemainingItemCount: 0,
		}

		options := &model.ListOptions{Limit: 10}
		mockPort.On("ListCertificate", ctx, options).Return(expectedResp, nil)

		resp, err := certService.ListCertificate(ctx, options)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.Items)
		mockPort.AssertExpectations(t)
	})

	t.Run("With pagination", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPort := new(MockCertificatePersistencePort)
		logger := slog.Default()

		certService := agentservice.NewCertificateService(mockPort, logger)

		certs := []*agentmodel.Certificate{
			{Metadata: agentmodel.CertificateMetadata{Name: "cert-1"}},
			{Metadata: agentmodel.CertificateMetadata{Name: "cert-2"}},
		}

		expectedResp := &model.ListResponse[*agentmodel.Certificate]{
			Items:              certs,
			Continue:           "next-page-token",
			RemainingItemCount: 5,
		}

		options := &model.ListOptions{Limit: 2, Continue: ""}
		mockPort.On("ListCertificate", ctx, options).Return(expectedResp, nil)

		resp, err := certService.ListCertificate(ctx, options)

		require.NoError(t, err)
		assert.Equal(t, "next-page-token", resp.Continue)
		assert.Equal(t, int64(5), resp.RemainingItemCount)
		mockPort.AssertExpectations(t)
	})

	t.Run("Persistence error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPort := new(MockCertificatePersistencePort)
		logger := slog.Default()

		certService := agentservice.NewCertificateService(mockPort, logger)

		options := &model.ListOptions{Limit: 10}
		mockPort.On("ListCertificate", ctx, options).Return(nil, errCertificatePersistence)

		resp, err := certService.ListCertificate(ctx, options)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to list certificates from persistence")
		mockPort.AssertExpectations(t)
	})
}

func TestCertificateService_SaveCertificate(t *testing.T) {
	t.Parallel()

	t.Run("Successfully save certificate", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPort := new(MockCertificatePersistencePort)
		logger := slog.Default()

		certService := agentservice.NewCertificateService(mockPort, logger)

		inputCert := &agentmodel.Certificate{
			Metadata: agentmodel.CertificateMetadata{Name: "new-cert"},
			Spec: agentmodel.CertificateSpec{
				Cert:       []byte("new-cert-data"),
				PrivateKey: []byte("new-key-data"),
			},
			Status: agentmodel.CertificateStatus{},
		}

		expectedCert := &agentmodel.Certificate{
			Metadata: agentmodel.CertificateMetadata{Name: "new-cert"},
			Spec: agentmodel.CertificateSpec{
				Cert:       []byte("new-cert-data"),
				PrivateKey: []byte("new-key-data"),
			},
			Status: agentmodel.CertificateStatus{},
		}

		mockPort.On("PutCertificate", ctx, inputCert).Return(expectedCert, nil)

		cert, err := certService.SaveCertificate(ctx, inputCert)

		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.Equal(t, "new-cert", cert.Metadata.Name)
		mockPort.AssertExpectations(t)
	})

	t.Run("Persistence error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPort := new(MockCertificatePersistencePort)
		logger := slog.Default()

		certService := agentservice.NewCertificateService(mockPort, logger)

		inputCert := &agentmodel.Certificate{
			Metadata: agentmodel.CertificateMetadata{Name: "new-cert"},
		}

		mockPort.On("PutCertificate", ctx, inputCert).Return(nil, errCertificatePersistence)

		cert, err := certService.SaveCertificate(ctx, inputCert)

		require.Error(t, err)
		assert.Nil(t, cert)
		assert.Contains(t, err.Error(), "failed to save certificate to persistence")
		mockPort.AssertExpectations(t)
	})
}

func TestCertificateService_DeleteCertificate(t *testing.T) {
	t.Parallel()

	t.Run("Successfully delete certificate", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPort := new(MockCertificatePersistencePort)
		logger := slog.Default()

		certService := agentservice.NewCertificateService(mockPort, logger)

		existingCert := &agentmodel.Certificate{
			Metadata: agentmodel.CertificateMetadata{Name: "cert-to-delete"},
			Spec:     agentmodel.CertificateSpec{Cert: []byte("cert-data")},
			Status:   agentmodel.CertificateStatus{Conditions: []model.Condition{}},
		}

		deletedAt := time.Now()
		deletedBy := "admin-user"

		updatedCert := &agentmodel.Certificate{
			Metadata: agentmodel.CertificateMetadata{
				Name:      "cert-to-delete",
				DeletedAt: deletedAt,
			},
			Spec: agentmodel.CertificateSpec{Cert: []byte("cert-data")},
			Status: agentmodel.CertificateStatus{
				Conditions: []model.Condition{
					{
						Type:               model.ConditionTypeDeleted,
						Status:             model.ConditionStatusTrue,
						LastTransitionTime: deletedAt,
						Reason:             deletedBy,
					},
				},
			},
		}

		mockPort.On("GetCertificate", ctx, "default", "cert-to-delete", (*model.GetOptions)(nil)).Return(existingCert, nil)
		mockPort.On("PutCertificate", ctx, mock.AnythingOfType("*agentmodel.Certificate")).Return(updatedCert, nil)

		cert, err := certService.DeleteCertificate(ctx, "default", "cert-to-delete", deletedAt, deletedBy)

		require.NoError(t, err)
		assert.NotNil(t, cert)
		mockPort.AssertExpectations(t)
	})

	t.Run("Certificate not found for deletion", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPort := new(MockCertificatePersistencePort)
		logger := slog.Default()

		certService := agentservice.NewCertificateService(mockPort, logger)

		mockPort.On("GetCertificate", ctx, "default", "non-existent", (*model.GetOptions)(nil)).
			Return(nil, port.ErrResourceNotExist)

		cert, err := certService.DeleteCertificate(ctx, "default", "non-existent", time.Now(), "admin")

		require.Error(t, err)
		assert.Nil(t, cert)
		assert.Contains(t, err.Error(), "failed to get certificate from persistence")
		mockPort.AssertExpectations(t)
	})

	t.Run("Error updating certificate during deletion", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPort := new(MockCertificatePersistencePort)
		logger := slog.Default()

		certService := agentservice.NewCertificateService(mockPort, logger)

		existingCert := &agentmodel.Certificate{
			Metadata: agentmodel.CertificateMetadata{Name: "cert-to-delete"},
			Spec:     agentmodel.CertificateSpec{},
			Status:   agentmodel.CertificateStatus{Conditions: []model.Condition{}},
		}

		mockPort.On("GetCertificate", ctx, "default", "cert-to-delete", (*model.GetOptions)(nil)).Return(existingCert, nil)
		mockPort.On(
			"PutCertificate", ctx, mock.AnythingOfType("*agentmodel.Certificate"),
		).Return(nil, errCertificatePersistence)

		cert, err := certService.DeleteCertificate(ctx, "default", "cert-to-delete", time.Now(), "admin")

		require.Error(t, err)
		assert.Nil(t, cert)
		assert.Contains(t, err.Error(), "failed to update certificate in persistence")
		mockPort.AssertExpectations(t)
	})
}

// Verify CertificateService implements CertificateUsecase interface.
var _ agentport.CertificateUsecase = (*agentservice.CertificateService)(nil)
