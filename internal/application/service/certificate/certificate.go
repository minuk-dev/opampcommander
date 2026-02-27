// Package certificate provides the CertificateService for managing certificates.
package certificate

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/helper"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/application/service/certificate/filter"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.CertificateManageUsecase = (*Service)(nil)

// Service is a service for managing certificates.
type Service struct {
	certificateUsecase domainport.CertificateUsecase
	mapper             *helper.Mapper
	sanityFilter       *filter.Sanity
	clock              clock.Clock
	logger             *slog.Logger
}

// NewCertificateService creates a new CertificateService.
func NewCertificateService(
	certificateUsecase domainport.CertificateUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		certificateUsecase: certificateUsecase,
		mapper:             helper.NewMapper(),
		sanityFilter:       filter.NewSanity(),
		clock:              clock.NewRealClock(),
		logger:             logger,
	}
}

// GetCertificate implements [port.CertificateManageUsecase].
func (s *Service) GetCertificate(
	ctx context.Context,
	name string,
) (*v1.Certificate, error) {
	certificate, err := s.certificateUsecase.GetCertificate(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get certificate: %w", err)
	}

	return s.mapper.MapCertificateToAPI(certificate), nil
}

// ListCertificates implements [port.CertificateManageUsecase].
func (s *Service) ListCertificates(
	ctx context.Context,
	options *model.ListOptions,
) (*v1.ListResponse[v1.Certificate], error) {
	certificates, err := s.certificateUsecase.ListCertificate(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("list certificates: %w", err)
	}

	return &v1.ListResponse[v1.Certificate]{
		Kind:       v1.CertificateKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           certificates.Continue,
			RemainingItemCount: certificates.RemainingItemCount,
		},
		Items: lo.Map(certificates.Items, func(item *model.Certificate, _ int) v1.Certificate {
			return *s.mapper.MapCertificateToAPI(item)
		}),
	}, nil
}

// CreateCertificate implements [port.CertificateManageUsecase].
func (s *Service) CreateCertificate(
	ctx context.Context,
	apiModel *v1.Certificate,
) (*v1.Certificate, error) {
	domainModel := s.mapper.MapAPIToCertificate(apiModel)

	now := s.clock.Now()

	createdBy, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn("failed to get user from context", slog.String("error", err.Error()))

		createdBy = security.NewAnonymousUser()
	}

	// Set the CreatedAt timestamp in metadata
	domainModel.Metadata.CreatedAt = now

	domainModel.Status.Conditions = append(domainModel.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeCreated,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: now,
		Reason:             createdBy.String(),
		Message:            "Certificate created",
	})

	created, err := s.certificateUsecase.SaveCertificate(ctx, domainModel)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}

	return s.mapper.MapCertificateToAPI(created), nil
}

// UpdateCertificate implements [port.CertificateManageUsecase].
func (s *Service) UpdateCertificate(
	ctx context.Context,
	name string,
	certificate *v1.Certificate,
) (*v1.Certificate, error) {
	existingDomainModel, err := s.certificateUsecase.GetCertificate(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get existing certificate: %w", err)
	}

	domainModel := s.mapper.MapAPIToCertificate(certificate)

	// Sanitize: preserve immutable fields from existing certificate
	domainModel = s.sanityFilter.Sanitize(existingDomainModel, domainModel)

	now := s.clock.Now()

	updatedBy, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn("failed to get user from context", slog.String("error", err.Error()))

		updatedBy = security.NewAnonymousUser()
	}

	domainModel.Status.Conditions = append(domainModel.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeUpdated,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: now,
		Reason:             updatedBy.String(),
		Message:            "Certificate updated",
	})

	updated, err := s.certificateUsecase.SaveCertificate(ctx, domainModel)
	if err != nil {
		return nil, fmt.Errorf("update certificate: %w", err)
	}

	return s.mapper.MapCertificateToAPI(updated), nil
}

// DeleteCertificate implements [port.CertificateManageUsecase].
func (s *Service) DeleteCertificate(
	ctx context.Context,
	name string,
) error {
	deletedBy, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn("failed to get user from context", slog.String("error", err.Error()))

		deletedBy = security.NewAnonymousUser()
	}

	_, err = s.certificateUsecase.DeleteCertificate(ctx, name, s.clock.Now(), deletedBy.String())
	if err != nil {
		return fmt.Errorf("delete certificate: %w", err)
	}

	return nil
}
