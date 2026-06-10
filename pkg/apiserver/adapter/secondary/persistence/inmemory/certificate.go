package inmemory

import (
	"context"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.CertificatePersistencePort = (*CertificateRepository)(nil)

// CertificateRepository is the in-memory implementation of
// [agentport.CertificatePersistencePort].
type CertificateRepository struct {
	store *store[namespacedName, *agentmodel.Certificate]
}

// NewCertificateRepository creates a new in-memory CertificateRepository.
func NewCertificateRepository() *CertificateRepository {
	return &CertificateRepository{
		store: newStore[namespacedName](func(cert *agentmodel.Certificate) *time.Time {
			return &cert.Metadata.DeletedAt
		}),
	}
}

// GetCertificate implements agentport.CertificatePersistencePort.
func (r *CertificateRepository) GetCertificate(
	_ context.Context, namespace string, name string, options *model.GetOptions,
) (*agentmodel.Certificate, error) {
	return r.store.get(namespacedName{Namespace: namespace, Name: name}, options)
}

// PutCertificate implements agentport.CertificatePersistencePort.
func (r *CertificateRepository) PutCertificate(
	_ context.Context, certificate *agentmodel.Certificate,
) (*agentmodel.Certificate, error) {
	r.store.put(namespacedName{
		Namespace: certificate.Metadata.Namespace,
		Name:      certificate.Metadata.Name,
	}, certificate)

	return certificate, nil
}

// ListCertificate implements agentport.CertificatePersistencePort.
func (r *CertificateRepository) ListCertificate(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Certificate], error) {
	return r.store.list(options, nil)
}
