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

var _ port.RolePersistencePort = (*RoleMongoAdapter)(nil)

const (
	roleCollectionName = "roles"
)

// RoleMongoAdapter is a struct that implements the RolePersistencePort interface.
type RoleMongoAdapter struct {
	common commonEntityAdapter[entity.Role, string]
}

// NewRoleRepository creates a new instance of RoleMongoAdapter.
func NewRoleRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *RoleMongoAdapter {
	collection := mongoDatabase.Collection(roleCollectionName)
	keyFunc := func(en *entity.Role) string {
		return en.Metadata.UID
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &RoleMongoAdapter{
		common: newCommonAdapter(
			logger,
			collection,
			entity.RoleKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetRole implements port.RolePersistencePort.
func (a *RoleMongoAdapter) GetRole(
	ctx context.Context, uid uuid.UUID,
) (*model.Role, error) {
	en, err := a.common.get(ctx, uid.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}

	return en.ToDomain(), nil
}

// GetRoleByName implements port.RolePersistencePort.
func (a *RoleMongoAdapter) GetRoleByName(
	ctx context.Context, displayName string,
) (*model.Role, error) {
	filter := bson.M{
		"spec.displayName":   displayName,
		"metadata.deletedAt": nil,
	}

	result := a.common.collection.FindOne(ctx, filter)

	err := result.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, port.ErrResourceNotExist
		}

		return nil, fmt.Errorf("get role by name: %w", err)
	}

	var en entity.Role

	err = result.Decode(&en)
	if err != nil {
		return nil, fmt.Errorf("decode role by name: %w", err)
	}

	return en.ToDomain(), nil
}

// PutRole implements port.RolePersistencePort.
func (a *RoleMongoAdapter) PutRole(
	ctx context.Context, role *model.Role,
) (*model.Role, error) {
	en := entity.RoleFromDomain(role)

	err := a.common.put(ctx, en)
	if err != nil {
		return nil, fmt.Errorf("put role: %w", err)
	}

	return role, nil
}

// ListRoles implements port.RolePersistencePort.
func (a *RoleMongoAdapter) ListRoles(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*model.Role], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*model.Role, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*model.Role]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// DeleteRole implements port.RolePersistencePort.
func (a *RoleMongoAdapter) DeleteRole(
	ctx context.Context, uid uuid.UUID,
) error {
	en, err := a.common.get(ctx, uid.String(), nil)
	if err != nil {
		return fmt.Errorf("get role for delete: %w", err)
	}

	domainRole := en.ToDomain()
	domainRole.Delete()

	deletedEn := entity.RoleFromDomain(domainRole)

	err = a.common.put(ctx, deletedEn)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}

	return nil
}
