package agentservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

var _ agentport.CertificateUsecase = (*CertificateService)(nil)

// CertificateService implements the CertificateUsecase interface.
type CertificateService struct {
	certificatePersistencePort agentport.CertificatePersistencePort
	logger                     *slog.Logger
}

// NewCertificateService creates a new instance of CertificateService.
func NewCertificateService(
	certificatePersistencePort agentport.CertificatePersistencePort,
	logger *slog.Logger,
) *CertificateService {
	return &CertificateService{
		certificatePersistencePort: certificatePersistencePort,
		logger:                     logger,
	}
}

// GetCertificate implements [agentport.CertificateUsecase].
func (c *CertificateService) GetCertificate(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.Certificate, error) {
	certificate, err := c.certificatePersistencePort.GetCertificate(ctx, namespace, name, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate from persistence: %w", err)
	}

	return certificate, nil
}

// ListCertificate implements [agentport.CertificateUsecase].
func (c *CertificateService) ListCertificate(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Certificate], error) {
	resp, err := c.certificatePersistencePort.ListCertificate(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates from persistence: %w", err)
	}

	return resp, nil
}

// SaveCertificate implements [agentport.CertificateUsecase].
func (c *CertificateService) SaveCertificate(
	ctx context.Context,
	certificate *agentmodel.Certificate,
) (*agentmodel.Certificate, error) {
	saved, err := c.certificatePersistencePort.PutCertificate(ctx, certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to save certificate to persistence: %w", err)
	}

	return saved, nil
}

// DeleteCertificate implements [agentport.CertificateUsecase].
func (c *CertificateService) DeleteCertificate(
	ctx context.Context,
	namespace string,
	name string,
	deletedAt time.Time,
	deletedBy string,
) (*agentmodel.Certificate, error) {
	certificate, err := c.certificatePersistencePort.GetCertificate(ctx, namespace, name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate from persistence: %w", err)
	}

	certificate.MarkAsDeleted(deletedAt, deletedBy)

	updatedCertificate, err := c.certificatePersistencePort.PutCertificate(ctx, certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to update certificate in persistence: %w", err)
	}

	return updatedCertificate, nil
}
