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
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ userport.PermissionPersistencePort = (*PermissionMongoAdapter)(nil)

const (
	permissionCollectionName = "permissions"
)

// PermissionMongoAdapter is a struct that implements the PermissionPersistencePort interface.
type PermissionMongoAdapter struct {
	common commonEntityAdapter[entity.Permission, string]
}

// NewPermissionRepository creates a new instance of PermissionMongoAdapter.
func NewPermissionRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *PermissionMongoAdapter {
	collection := mongoDatabase.Collection(permissionCollectionName)
	keyFunc := func(en *entity.Permission) string {
		return en.Metadata.UID
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &PermissionMongoAdapter{
		common: newCommonAdapter(
			logger,
			collection,
			entity.PermissionKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetPermission implements userport.PermissionPersistencePort.
func (a *PermissionMongoAdapter) GetPermission(
	ctx context.Context, uid uuid.UUID,
) (*usermodel.Permission, error) {
	en, err := a.common.get(ctx, uid.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("get permission: %w", err)
	}

	return en.ToDomain(), nil
}

// GetPermissionByName implements userport.PermissionPersistencePort.
func (a *PermissionMongoAdapter) GetPermissionByName(
	ctx context.Context, name string,
) (*usermodel.Permission, error) {
	filter := bson.M{
		"spec.name":          name,
		"metadata.deletedAt": nil,
	}

	result := a.common.collection.FindOne(ctx, filter)

	err := result.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, port.ErrResourceNotExist
		}

		return nil, fmt.Errorf("get permission by name: %w", err)
	}

	var permEntity entity.Permission

	err = result.Decode(&permEntity)
	if err != nil {
		return nil, fmt.Errorf("decode permission by name: %w", err)
	}

	return permEntity.ToDomain(), nil
}

// PutPermission implements userport.PermissionPersistencePort.
func (a *PermissionMongoAdapter) PutPermission(
	ctx context.Context, permission *usermodel.Permission,
) (*usermodel.Permission, error) {
	en := entity.PermissionFromDomain(permission)

	err := a.common.put(ctx, en)
	if err != nil {
		return nil, fmt.Errorf("put permission: %w", err)
	}

	return permission, nil
}

// ListPermissions implements userport.PermissionPersistencePort.
func (a *PermissionMongoAdapter) ListPermissions(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*usermodel.Permission], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*usermodel.Permission, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*usermodel.Permission]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// DeletePermission implements userport.PermissionPersistencePort.
func (a *PermissionMongoAdapter) DeletePermission(
	ctx context.Context, uid uuid.UUID,
) error {
	en, err := a.common.get(ctx, uid.String(), nil)
	if err != nil {
		return fmt.Errorf("get permission for delete: %w", err)
	}

	domainPermission := en.ToDomain()
	domainPermission.Delete()

	deletedEn := entity.PermissionFromDomain(domainPermission)

	err = a.common.put(ctx, deletedEn)
	if err != nil {
		return fmt.Errorf("delete permission: %w", err)
	}

	return nil
}
