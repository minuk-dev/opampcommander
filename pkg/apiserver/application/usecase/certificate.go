package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// CertificateManageUsecase manages the TLS certificates the server stores
// for securing agent connections. It backs the /api/v1/certificates
// controller.
type CertificateManageUsecase interface {
	// GetCertificate returns the named certificate in namespace, or
	// model.ErrResourceNotExist if absent.
	GetCertificate(ctx context.Context, namespace string, name string,
		options *port.GetOptions) (*v1.Certificate, error)
	// ListCertificates returns a paged list of certificates across namespaces.
	ListCertificates(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.Certificate], error)
	// CreateCertificate persists a new certificate, returning
	// model.ErrResourceAlreadyExist on a duplicate.
	CreateCertificate(ctx context.Context, certificate *v1.Certificate) (*v1.Certificate, error)
	// UpdateCertificate replaces the named certificate;
	// optimistic-concurrency controlled (model.ErrConflict on a stale write).
	UpdateCertificate(ctx context.Context, namespace string, name string,
		certificate *v1.Certificate) (*v1.Certificate, error)
	// DeleteCertificate removes the named certificate.
	DeleteCertificate(ctx context.Context, namespace string, name string) error
}
