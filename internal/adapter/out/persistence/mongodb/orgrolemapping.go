//nolint:dupl // MongoDB adapter pattern - similar structure is intentional
package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.OrgRoleMappingPersistencePort = (*OrgRoleMappingMongoAdapter)(nil)

const (
	orgRoleMappingCollectionName = "org_role_mappings"
)

// OrgRoleMappingMongoAdapter is a struct that implements the OrgRoleMappingPersistencePort interface.
type OrgRoleMappingMongoAdapter struct {
	common commonEntityAdapter[entity.OrgRoleMapping, string]
	logger *slog.Logger
}

// NewOrgRoleMappingRepository creates a new instance of OrgRoleMappingMongoAdapter.
func NewOrgRoleMappingRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *OrgRoleMappingMongoAdapter {
	collection := mongoDatabase.Collection(orgRoleMappingCollectionName)
	keyFunc := func(en *entity.OrgRoleMapping) string {
		return en.Metadata.UID
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &OrgRoleMappingMongoAdapter{
		common: newCommonAdapter(
			logger,
			collection,
			entity.OrgRoleMappingKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
		logger: logger,
	}
}

// GetOrgRoleMapping implements port.OrgRoleMappingPersistencePort.
func (a *OrgRoleMappingMongoAdapter) GetOrgRoleMapping(
	ctx context.Context, uid uuid.UUID,
) (*model.OrgRoleMapping, error) {
	en, err := a.common.get(ctx, uid.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("get org role mapping: %w", err)
	}

	return en.ToDomain(), nil
}

// PutOrgRoleMapping implements port.OrgRoleMappingPersistencePort.
func (a *OrgRoleMappingMongoAdapter) PutOrgRoleMapping(
	ctx context.Context, mapping *model.OrgRoleMapping,
) (*model.OrgRoleMapping, error) {
	en := entity.OrgRoleMappingFromDomain(mapping)

	err := a.common.put(ctx, en)
	if err != nil {
		return nil, fmt.Errorf("put org role mapping: %w", err)
	}

	return mapping, nil
}

// ListOrgRoleMappings implements port.OrgRoleMappingPersistencePort.
func (a *OrgRoleMappingMongoAdapter) ListOrgRoleMappings(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*model.OrgRoleMapping], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*model.OrgRoleMapping, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*model.OrgRoleMapping]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// ListOrgRoleMappingsByProvider implements port.OrgRoleMappingPersistencePort.
func (a *OrgRoleMappingMongoAdapter) ListOrgRoleMappingsByProvider(
	ctx context.Context, provider string,
) ([]*model.OrgRoleMapping, error) {
	filter := bson.M{
		"spec.provider":      provider,
		"metadata.deletedAt": nil,
	}

	cursor, err := a.common.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find org role mappings by provider: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			a.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var entities []entity.OrgRoleMapping

	err = cursor.All(ctx, &entities)
	if err != nil {
		return nil, fmt.Errorf("decode org role mappings by provider: %w", err)
	}

	mappings := make([]*model.OrgRoleMapping, 0, len(entities))
	for i := range entities {
		mappings = append(mappings, entities[i].ToDomain())
	}

	return mappings, nil
}

// DeleteOrgRoleMapping implements port.OrgRoleMappingPersistencePort.
func (a *OrgRoleMappingMongoAdapter) DeleteOrgRoleMapping(
	ctx context.Context, uid uuid.UUID,
) error {
	en, err := a.common.get(ctx, uid.String(), nil)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return port.ErrResourceNotExist
		}

		return fmt.Errorf("get org role mapping for delete: %w", err)
	}

	domainMapping := en.ToDomain()
	domainMapping.Delete()

	deletedEn := entity.OrgRoleMappingFromDomain(domainMapping)

	err = a.common.put(ctx, deletedEn)
	if err != nil {
		return fmt.Errorf("delete org role mapping: %w", err)
	}

	return nil
}
