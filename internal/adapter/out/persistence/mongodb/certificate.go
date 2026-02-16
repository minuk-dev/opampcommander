package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.CertificatePersistencePort = (*CertificateMongoAdapter)(nil)

const (
	certificateCollectionName = "certificates"
)

// CertificateMongoAdapter is a struct that implements the CertificatePersistencePort interface.
type CertificateMongoAdapter struct {
	common commonEntityAdapter[entity.Certificate, string]
}

// NewCertificateRepository creates a new instance of CertificateMongoAdapter.
func NewCertificateRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *CertificateMongoAdapter {
	collection := mongoDatabase.Collection(certificateCollectionName)
	keyFunc := func(en *entity.Certificate) string {
		return en.Metadata.Name
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &CertificateMongoAdapter{
		common: newCommonAdapter(
			logger,
			collection,
			entity.CertificateKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetCertificate implements port.CertificatePersistencePort.
func (c *CertificateMongoAdapter) GetCertificate(
	ctx context.Context, name string,
) (*model.Certificate, error) {
	en, err := c.common.get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get certificate: %w", err)
	}

	return en.ToDomain(), nil
}

// ListCertificate implements port.CertificatePersistencePort.
func (c *CertificateMongoAdapter) ListCertificate(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*model.Certificate], error) {
	resp, err := c.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*model.Certificate, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*model.Certificate]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutCertificate implements port.CertificatePersistencePort.
func (c *CertificateMongoAdapter) PutCertificate(
	ctx context.Context, certificate *model.Certificate,
) (*model.Certificate, error) {
	en := entity.CertificateFromDomain(certificate)

	err := c.common.put(ctx, en)
	if err != nil {
		return nil, fmt.Errorf("put certificate: %w", err)
	}

	// Return the domain model directly instead of querying again
	// This avoids issues with soft-deleted documents not being found
	return certificate, nil
}
