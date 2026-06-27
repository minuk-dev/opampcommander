package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// CertificateManageUsecase is a use case that handles certificate management operations.
type CertificateManageUsecase interface {
	GetCertificate(ctx context.Context, namespace string, name string,
		options *port.GetOptions) (*v1.Certificate, error)
	ListCertificates(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.Certificate], error)
	CreateCertificate(ctx context.Context, certificate *v1.Certificate) (*v1.Certificate, error)
	UpdateCertificate(ctx context.Context, namespace string, name string,
		certificate *v1.Certificate) (*v1.Certificate, error)
	DeleteCertificate(ctx context.Context, namespace string, name string) error
}
