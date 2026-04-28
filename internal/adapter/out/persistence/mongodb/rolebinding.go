package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ userport.RoleBindingPersistencePort = (*RoleBindingMongoAdapter)(nil)

const (
	roleBindingCollectionName     = "rolebindings"
	roleBindingNamespaceFieldName = "metadata.namespace"
	roleBindingNameFieldName      = "metadata.name"
	roleBindingDeletedAtFieldName = "metadata.deletedAt"
)

// RoleBindingMongoAdapter implements the RoleBindingPersistencePort interface.
type RoleBindingMongoAdapter struct {
	common commonEntityAdapter[entity.RoleBinding, string]
	logger *slog.Logger
}

// NewRoleBindingRepository creates a new instance of RoleBindingMongoAdapter.
func NewRoleBindingRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *RoleBindingMongoAdapter {
	collection := mongoDatabase.Collection(roleBindingCollectionName)
	keyFunc := func(en *entity.RoleBinding) string {
		return en.Metadata.Name
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &RoleBindingMongoAdapter{
		common: newCommonAdapter(
			logger,
			collection,
			entity.RoleBindingKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
		logger: logger,
	}
}

// GetRoleBinding implements userport.RoleBindingPersistencePort.
func (a *RoleBindingMongoAdapter) GetRoleBinding(
	ctx context.Context,
	namespace, name string,
) (*usermodel.RoleBinding, error) {
	filter := a.filterByNamespaceAndNameExcludingDeleted(namespace, name)

	result := a.common.collection.FindOne(ctx, filter)

	err := result.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, port.ErrResourceNotExist
		}

		return nil, fmt.Errorf("get role binding: %w", err)
	}

	var roleBindingEntity entity.RoleBinding

	err = result.Decode(&roleBindingEntity)
	if err != nil {
		return nil, fmt.Errorf("decode role binding: %w", err)
	}

	return roleBindingEntity.ToDomain(), nil
}

// PutRoleBinding implements userport.RoleBindingPersistencePort.
func (a *RoleBindingMongoAdapter) PutRoleBinding(
	ctx context.Context,
	roleBinding *usermodel.RoleBinding,
) (*usermodel.RoleBinding, error) {
	roleBindingEntity := entity.RoleBindingFromDomain(roleBinding)

	// Use a compound (namespace, name) filter so bindings with the same name
	// in different namespaces are stored as separate documents.
	filter := a.filterByNamespaceAndName(roleBindingEntity.Metadata.Namespace, roleBindingEntity.Metadata.Name)

	_, err := a.common.collection.ReplaceOne(ctx, filter, roleBindingEntity, options.Replace().SetUpsert(true))
	if err != nil {
		return nil, fmt.Errorf("put role binding: %w", err)
	}

	if roleBinding.IsDeleted() {
		return roleBinding, nil
	}

	return a.GetRoleBinding(ctx, roleBinding.Metadata.Namespace, roleBinding.Metadata.Name)
}

// ListRoleBindings implements userport.RoleBindingPersistencePort.
func (a *RoleBindingMongoAdapter) ListRoleBindings(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*usermodel.RoleBinding], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*usermodel.RoleBinding, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*usermodel.RoleBinding]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// DeleteRoleBinding implements userport.RoleBindingPersistencePort.
// Uses a single atomic UpdateOne to avoid read-modify-write races.
func (a *RoleBindingMongoAdapter) DeleteRoleBinding(
	ctx context.Context,
	namespace, name string,
) error {
	filter := a.filterByNamespaceAndNameExcludingDeleted(namespace, name)
	update := bson.M{"$set": bson.M{roleBindingDeletedAtFieldName: time.Now()}}

	result, err := a.common.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("soft-delete role binding: %w", err)
	}

	if result.MatchedCount == 0 {
		return port.ErrResourceNotExist
	}

	return nil
}

func (a *RoleBindingMongoAdapter) filterByNamespaceAndName(namespace, name string) bson.M {
	return bson.M{
		roleBindingNamespaceFieldName: sanitizeResourceName(namespace),
		roleBindingNameFieldName:      sanitizeResourceName(name),
	}
}

func (a *RoleBindingMongoAdapter) filterByNamespaceAndNameExcludingDeleted(namespace, name string) bson.M {
	filter := a.filterByNamespaceAndName(namespace, name)
	filter[roleBindingDeletedAtFieldName] = nil

	return filter
}
