// Package certificate provides the CertificateService for managing certificates.
package certificate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ usecase.CertificateManageUsecase = (*Service)(nil)

// Service is a service for managing certificates. It maps between the HTTP DTOs
// and the domain, resolves the acting user, and delegates all lifecycle rules
// (stamping, immutable-field preservation) to the domain CertificateUsecase.
type Service struct {
	certificateUsecase agentport.CertificateUsecase
	mapper             *helper.Mapper
	clock              clock.Clock
	logger             *slog.Logger
	issuer             *Issuer
}

// NewCertificateService creates a new CertificateService.
func NewCertificateService(
	certificateUsecase agentport.CertificateUsecase,
	logger *slog.Logger,
	issuer *Issuer,
) *Service {
	realClock := clock.NewRealClock()

	return &Service{
		certificateUsecase: certificateUsecase,
		mapper:             helper.NewMapper(realClock, 0),
		clock:              realClock,
		logger:             logger,
		issuer:             issuer,
	}
}

// IssueClientCertificate signs and stores an agent-specific client certificate.
func (s *Service) IssueClientCertificate(
	ctx context.Context, namespace string, request *v1.IssueClientCertificateRequest,
) (*v1.Certificate, error) {
	if s.issuer == nil || !s.issuer.Enabled() {
		return nil, ErrIssuerUnavailable
	}

	_, err := s.certificateUsecase.GetCertificate(ctx, namespace, request.Name, nil)
	if err == nil {
		return nil, model.ErrResourceAlreadyExist
	}

	if !errors.Is(err, model.ErrResourceNotExist) {
		return nil, fmt.Errorf("check certificate name: %w", err)
	}

	cert, key, err := s.issuer.Issue(request.InstanceUID)
	if err != nil {
		return nil, fmt.Errorf("issue client certificate: %w", err)
	}

	//exhaustruct:ignore // Creation metadata and status are stamped by the domain.
	return s.CreateCertificate(ctx, &v1.Certificate{
		Kind: v1.CertificateKind, APIVersion: v1.APIVersion,
		//exhaustruct:ignore // Creation metadata is stamped by the domain.
		Metadata: v1.CertificateMetadata{Name: request.Name, Namespace: namespace},
		//exhaustruct:ignore // The agent's server-trust CA is independent of the client issuer.
		Spec: v1.CertificateSpec{Cert: string(cert), PrivateKey: string(key)},
	})
}

// GetCertificate implements [usecase.CertificateManageUsecase].
func (s *Service) GetCertificate(
	ctx context.Context,
	namespace string,
	name string,
	options *port.GetOptions,
) (*v1.Certificate, error) {
	certificate, err := s.certificateUsecase.GetCertificate(ctx, namespace, name, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("get certificate: %w", err)
	}

	return s.mapper.MapCertificateToAPI(certificate), nil
}

// ListCertificates implements [usecase.CertificateManageUsecase].
func (s *Service) ListCertificates(
	ctx context.Context,
	namespace string,
	options *port.ListOptions,
) (*v1.ListResponse[v1.Certificate], error) {
	certificates, err := s.certificateUsecase.ListCertificate(ctx, namespace, options.ToDomain())
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
		Items: lo.Map(certificates.Items, func(item *agentmodel.Certificate, _ int) v1.Certificate {
			return *s.mapper.MapCertificateToAPI(item)
		}),
	}, nil
}

// CreateCertificate implements [usecase.CertificateManageUsecase].
func (s *Service) CreateCertificate(
	ctx context.Context,
	apiModel *v1.Certificate,
) (*v1.Certificate, error) {
	domainModel := s.mapper.MapAPIToCertificate(apiModel)

	created, err := s.certificateUsecase.CreateCertificate(ctx, domainModel, s.actor(ctx))
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}

	return s.mapper.MapCertificateToAPI(created), nil
}

// UpdateCertificate implements [usecase.CertificateManageUsecase].
func (s *Service) UpdateCertificate(
	ctx context.Context,
	namespace string,
	name string,
	certificate *v1.Certificate,
) (*v1.Certificate, error) {
	domainModel := s.mapper.MapAPIToCertificate(certificate)

	updated, err := s.certificateUsecase.UpdateCertificate(ctx, namespace, name, domainModel, s.actor(ctx))
	if err != nil {
		return nil, fmt.Errorf("update certificate: %w", err)
	}

	return s.mapper.MapCertificateToAPI(updated), nil
}

// DeleteCertificate implements [usecase.CertificateManageUsecase].
func (s *Service) DeleteCertificate(
	ctx context.Context,
	namespace string,
	name string,
) error {
	_, err := s.certificateUsecase.DeleteCertificate(
		ctx, namespace, name, s.clock.Now(), s.actor(ctx),
	)
	if err != nil {
		return fmt.Errorf("delete certificate: %w", err)
	}

	return nil
}

// actor resolves the acting user from the request context, falling back to an
// anonymous identity (and logging) when none is present.
func (s *Service) actor(ctx context.Context) string {
	user, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn("failed to get user from context", slog.String("error", err.Error()))

		user = security.NewAnonymousUser()
	}

	return user.String()
}
