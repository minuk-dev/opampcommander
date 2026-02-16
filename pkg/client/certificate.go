//nolint:dupl // Similar structure to other resource services is intentional
package client

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListCertificateURL is the path to list all certificates.
	ListCertificateURL = "/api/v1/certificates"
	// GetCertificateURL is the path to get a certificate by name.
	GetCertificateURL = "/api/v1/certificates/{id}"
	// CreateCertificateURL is the path to create a new certificate.
	CreateCertificateURL = "/api/v1/certificates"
	// UpdateCertificateURL is the path to update an existing certificate.
	UpdateCertificateURL = "/api/v1/certificates/{id}"
	// DeleteCertificateURL is the path to delete a certificate.
	DeleteCertificateURL = "/api/v1/certificates/{id}"
)

// CertificateService provides methods to interact with certificates.
type CertificateService struct {
	service *service
}

// NewCertificateService creates a new CertificateService.
func NewCertificateService(service *service) *CertificateService {
	return &CertificateService{
		service: service,
	}
}

// GetCertificate retrieves a certificate by its name.
func (s *CertificateService) GetCertificate(
	ctx context.Context,
	name string,
) (*v1.Certificate, error) {
	return getResource[v1.Certificate](ctx, s.service, GetCertificateURL, name)
}

// CertificateListResponse represents a list of certificates with metadata.
type CertificateListResponse = v1.ListResponse[v1.Certificate]

// ListCertificates lists all certificates.
func (s *CertificateService) ListCertificates(
	ctx context.Context,
	opts ...ListOption,
) (*CertificateListResponse, error) {
	var listSettings ListSettings
	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	return listResources[v1.Certificate](
		ctx,
		s.service,
		ListCertificateURL,
		ListSettings{
			limit:         listSettings.limit,
			continueToken: listSettings.continueToken,
		},
	)
}

// CreateCertificate creates a new certificate.
func (s *CertificateService) CreateCertificate(
	ctx context.Context,
	createRequest *v1.Certificate,
) (*v1.Certificate, error) {
	return createResource[v1.Certificate, v1.Certificate](
		ctx,
		s.service,
		CreateCertificateURL,
		createRequest,
	)
}

// UpdateCertificate updates an existing certificate.
func (s *CertificateService) UpdateCertificate(
	ctx context.Context,
	updateRequest *v1.Certificate,
) (*v1.Certificate, error) {
	return updateResource(
		ctx,
		s.service,
		UpdateCertificateURL,
		updateRequest,
	)
}

// DeleteCertificate deletes a certificate by its name.
func (s *CertificateService) DeleteCertificate(ctx context.Context, name string) error {
	return deleteResource(ctx, s.service, DeleteCertificateURL, name)
}
