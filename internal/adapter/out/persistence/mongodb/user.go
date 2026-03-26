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

var _ port.UserPersistencePort = (*UserMongoAdapter)(nil)

const (
	userCollectionName = "users"
)

// UserMongoAdapter is a struct that implements the UserPersistencePort interface.
type UserMongoAdapter struct {
	common commonEntityAdapter[entity.User, string]
}

// NewUserRepository creates a new instance of UserMongoAdapter.
func NewUserRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *UserMongoAdapter {
	collection := mongoDatabase.Collection(userCollectionName)
	keyFunc := func(en *entity.User) string {
		return en.Metadata.UID
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &UserMongoAdapter{
		common: newCommonAdapter(
			logger,
			collection,
			entity.UserKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetUser implements port.UserPersistencePort.
func (a *UserMongoAdapter) GetUser(
	ctx context.Context, uid uuid.UUID,
) (*model.User, error) {
	en, err := a.common.get(ctx, uid.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return en.ToDomain(), nil
}

// GetUserByEmail implements port.UserPersistencePort.
func (a *UserMongoAdapter) GetUserByEmail(
	ctx context.Context, email string,
) (*model.User, error) {
	filter := bson.M{
		"spec.email":       email,
		"metadata.deletedAt": nil,
	}

	result := a.common.collection.FindOne(ctx, filter)

	err := result.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, port.ErrResourceNotExist
		}

		return nil, fmt.Errorf("get user by email: %w", err)
	}

	var en entity.User

	err = result.Decode(&en)
	if err != nil {
		return nil, fmt.Errorf("decode user by email: %w", err)
	}

	return en.ToDomain(), nil
}

// PutUser implements port.UserPersistencePort.
func (a *UserMongoAdapter) PutUser(
	ctx context.Context, user *model.User,
) (*model.User, error) {
	en := entity.UserFromDomain(user)

	err := a.common.put(ctx, en)
	if err != nil {
		return nil, fmt.Errorf("put user: %w", err)
	}

	return user, nil
}

// ListUsers implements port.UserPersistencePort.
func (a *UserMongoAdapter) ListUsers(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*model.User], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*model.User, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*model.User]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// DeleteUser implements port.UserPersistencePort.
func (a *UserMongoAdapter) DeleteUser(
	ctx context.Context, uid uuid.UUID,
) error {
	en, err := a.common.get(ctx, uid.String(), nil)
	if err != nil {
		return fmt.Errorf("get user for delete: %w", err)
	}

	domainUser := en.ToDomain()
	domainUser.Delete()

	deletedEn := entity.UserFromDomain(domainUser)

	err = a.common.put(ctx, deletedEn)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}
