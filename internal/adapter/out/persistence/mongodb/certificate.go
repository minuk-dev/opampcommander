//nolint:dupl // MongoDB adapter pattern - similar structure is intentional
package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ agentport.CertificatePersistencePort = (*CertificateMongoAdapter)(nil)

const (
	certificateCollectionName     = "certificates"
	certificateNamespaceFieldName = "metadata.namespace"
	certificateNameFieldName      = "metadata.name"
	certificateDeletedAtFieldName = "metadata.deletedAt"
)

// CertificateMongoAdapter is a struct that implements the CertificatePersistencePort interface.
type CertificateMongoAdapter struct {
	collection *mongo.Collection
	common     commonEntityAdapter[entity.Certificate, string]
	logger     *slog.Logger
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
		collection: collection,
		logger:     logger,
		common: newCommonAdapter(
			logger,
			collection,
			entity.CertificateNameFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetCertificate implements agentport.CertificatePersistencePort.
func (c *CertificateMongoAdapter) GetCertificate(
	ctx context.Context, namespace string, name string,
) (*agentmodel.Certificate, error) {
	filter := c.filterByNamespaceAndNameExcludingDeleted(namespace, name)

	result := c.collection.FindOne(ctx, filter)

	err := result.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, port.ErrResourceNotExist
		}

		return nil, fmt.Errorf("get certificate: %w", err)
	}

	var certificateEntity entity.Certificate

	err = result.Decode(&certificateEntity)
	if err != nil {
		return nil, fmt.Errorf("decode certificate: %w", err)
	}

	return certificateEntity.ToDomain(), nil
}

// ListCertificate implements agentport.CertificatePersistencePort.
func (c *CertificateMongoAdapter) ListCertificate(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Certificate], error) {
	resp, err := c.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*agentmodel.Certificate, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*agentmodel.Certificate]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutCertificate implements agentport.CertificatePersistencePort.
func (c *CertificateMongoAdapter) PutCertificate(
	ctx context.Context, certificate *agentmodel.Certificate,
) (*agentmodel.Certificate, error) {
	certificateEntity := entity.CertificateFromDomain(certificate)
	namespace := certificate.Metadata.Namespace
	name := certificate.Metadata.Name

	_, err := c.collection.ReplaceOne(ctx,
		c.filterByNamespaceAndName(namespace, name),
		certificateEntity,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		return nil, fmt.Errorf("put certificate: %w", err)
	}

	// Return the domain model directly instead of querying again
	// This avoids issues with soft-deleted documents not being found
	return certificate, nil
}

func (c *CertificateMongoAdapter) filterByNamespaceAndName(
	namespace, name string,
) bson.M {
	return bson.M{
		certificateNamespaceFieldName: sanitizeResourceName(namespace),
		certificateNameFieldName:      sanitizeResourceName(name),
	}
}

func (c *CertificateMongoAdapter) filterByNamespaceAndNameExcludingDeleted(
	namespace, name string,
) bson.M {
	filter := c.filterByNamespaceAndName(namespace, name)
	filter[certificateDeletedAtFieldName] = nil

	return filter
}
