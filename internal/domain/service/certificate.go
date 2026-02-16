package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.CertificateUsecase = (*CertificateService)(nil)

type CertificateService struct {
	certificatePersistencePort port.CertificatePersistencePort
	logger                     *slog.Logger
}

func NewCertificateService(
	certificatePersistencePort port.CertificatePersistencePort,
	logger *slog.Logger,
) *CertificateService {
	return &CertificateService{
		certificatePersistencePort: certificatePersistencePort,
		logger:                     logger,
	}
}

// GetCertificate implements [port.CertificateUsecase].
func (c *CertificateService) GetCertificate(
	ctx context.Context,
	name string,
) (*model.Certificate, error) {
	certificate, err := c.certificatePersistencePort.GetCertificate(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate from persistence: %w", err)
	}

	return certificate, nil
}

// ListCertificate implements [port.CertificateUsecase].
func (c *CertificateService) ListCertificate(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Certificate], error) {
	resp, err := c.certificatePersistencePort.ListCertificate(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates from persistence: %w", err)
	}

	return resp, nil
}

// SaveCertificate implements [port.CertificateUsecase].
func (c *CertificateService) SaveCertificate(
	ctx context.Context,
	certificate *model.Certificate,
) (*model.Certificate, error) {
	saved, err := c.certificatePersistencePort.PutCertificate(ctx, certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to save certificate to persistence: %w", err)
	}

	return saved, nil
}

// DeleteCertificate implements [port.CertificateUsecase].
func (c *CertificateService) DeleteCertificate(
	ctx context.Context,
	name string,
	deletedAt time.Time,
	deletedBy string,
) (*model.Certificate, error) {
	certificate, err := c.certificatePersistencePort.GetCertificate(ctx, name)
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
