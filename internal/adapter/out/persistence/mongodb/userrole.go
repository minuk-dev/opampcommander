//nolint:dupl // MongoDB adapter pattern - similar query structures are intentional.
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

var _ userport.UserRolePersistencePort = (*UserRoleMongoAdapter)(nil)

const (
	userRoleCollectionName = "user_roles"
)

// UserRoleMongoAdapter is a struct that implements the UserRolePersistencePort interface.
type UserRoleMongoAdapter struct {
	common         commonEntityAdapter[entity.UserRole, string]
	roleCollection *mongo.Collection
	userCollection *mongo.Collection
	logger         *slog.Logger
}

// NewUserRoleRepository creates a new instance of UserRoleMongoAdapter.
func NewUserRoleRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *UserRoleMongoAdapter {
	collection := mongoDatabase.Collection(userRoleCollectionName)
	keyFunc := func(en *entity.UserRole) string {
		return en.Metadata.UID
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &UserRoleMongoAdapter{
		common: newCommonAdapter(
			logger,
			collection,
			entity.UserRoleKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
		roleCollection: mongoDatabase.Collection(roleCollectionName),
		userCollection: mongoDatabase.Collection(userCollectionName),
		logger:         logger,
	}
}

// GetUserRole implements userport.UserRolePersistencePort.
func (a *UserRoleMongoAdapter) GetUserRole(
	ctx context.Context, uid uuid.UUID,
) (*usermodel.UserRole, error) {
	en, err := a.common.get(ctx, uid.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("get user role: %w", err)
	}

	return en.ToDomain(), nil
}

// GetUserRoleByUserAndRole implements userport.UserRolePersistencePort.
func (a *UserRoleMongoAdapter) GetUserRoleByUserAndRole(
	ctx context.Context, userID, roleID uuid.UUID, namespace string,
) (*usermodel.UserRole, error) {
	filter := bson.M{
		"spec.userID":        userID.String(),
		"spec.roleID":        roleID.String(),
		"spec.namespace":     namespace,
		"metadata.deletedAt": nil,
	}

	var userRoleEntity entity.UserRole

	err := a.common.collection.FindOne(ctx, filter).Decode(&userRoleEntity)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, port.ErrResourceNotExist
		}

		return nil, fmt.Errorf("find user role by user and role: %w", err)
	}

	return userRoleEntity.ToDomain(), nil
}

// PutUserRole implements userport.UserRolePersistencePort.
func (a *UserRoleMongoAdapter) PutUserRole(
	ctx context.Context, userRole *usermodel.UserRole,
) (*usermodel.UserRole, error) {
	en := entity.UserRoleFromDomain(userRole)

	err := a.common.put(ctx, en)
	if err != nil {
		return nil, fmt.Errorf("put user role: %w", err)
	}

	return userRole, nil
}

// ListUserRoles implements userport.UserRolePersistencePort.
func (a *UserRoleMongoAdapter) ListUserRoles(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*usermodel.UserRole], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*usermodel.UserRole, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*usermodel.UserRole]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// GetUserRoles implements userport.UserRolePersistencePort.
func (a *UserRoleMongoAdapter) GetUserRoles(
	ctx context.Context, userID uuid.UUID,
) ([]*usermodel.Role, error) {
	roleIDs, err := a.findRoleIDsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(roleIDs) == 0 {
		return []*usermodel.Role{}, nil
	}

	filter := bson.M{
		"metadata.uid":       bson.M{"$in": roleIDs},
		"metadata.deletedAt": nil,
	}

	cursor, err := a.roleCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find roles for user: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			a.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var roleEntities []entity.Role

	err = cursor.All(ctx, &roleEntities)
	if err != nil {
		return nil, fmt.Errorf("decode roles for user: %w", err)
	}

	roles := make([]*usermodel.Role, 0, len(roleEntities))
	for i := range roleEntities {
		roles = append(roles, roleEntities[i].ToDomain())
	}

	return roles, nil
}

// GetUserRolesInNamespace implements userport.UserRolePersistencePort.
func (a *UserRoleMongoAdapter) GetUserRolesInNamespace(
	ctx context.Context, userID uuid.UUID, namespace string,
) ([]*usermodel.Role, error) {
	roleIDs, err := a.findRoleIDsByUserIDAndNamespace(ctx, userID, namespace)
	if err != nil {
		return nil, err
	}

	if len(roleIDs) == 0 {
		return []*usermodel.Role{}, nil
	}

	filter := bson.M{
		"metadata.uid":       bson.M{"$in": roleIDs},
		"metadata.deletedAt": nil,
	}

	cursor, err := a.roleCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find roles for user in namespace: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			a.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var roleEntities []entity.Role

	err = cursor.All(ctx, &roleEntities)
	if err != nil {
		return nil, fmt.Errorf("decode roles for user in namespace: %w", err)
	}

	roles := make([]*usermodel.Role, 0, len(roleEntities))
	for i := range roleEntities {
		roles = append(roles, roleEntities[i].ToDomain())
	}

	return roles, nil
}

// GetRoleUsers implements userport.UserRolePersistencePort.
func (a *UserRoleMongoAdapter) GetRoleUsers(
	ctx context.Context, roleID uuid.UUID,
) ([]*usermodel.User, error) {
	userIDs, err := a.findUserIDsByRoleID(ctx, roleID)
	if err != nil {
		return nil, err
	}

	if len(userIDs) == 0 {
		return []*usermodel.User{}, nil
	}

	filter := bson.M{
		"metadata.uid":       bson.M{"$in": userIDs},
		"metadata.deletedAt": nil,
	}

	cursor, err := a.userCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find users for role: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			a.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var userEntities []entity.User

	err = cursor.All(ctx, &userEntities)
	if err != nil {
		return nil, fmt.Errorf("decode users for role: %w", err)
	}

	users := make([]*usermodel.User, 0, len(userEntities))
	for i := range userEntities {
		users = append(users, userEntities[i].ToDomain())
	}

	return users, nil
}

// DeleteUserRole implements userport.UserRolePersistencePort.
func (a *UserRoleMongoAdapter) DeleteUserRole(
	ctx context.Context, uid uuid.UUID,
) error {
	en, err := a.common.get(ctx, uid.String(), nil)
	if err != nil {
		return fmt.Errorf("get user role for delete: %w", err)
	}

	domainUserRole := en.ToDomain()
	domainUserRole.Delete()

	deletedEn := entity.UserRoleFromDomain(domainUserRole)

	err = a.common.put(ctx, deletedEn)
	if err != nil {
		return fmt.Errorf("delete user role: %w", err)
	}

	return nil
}

// DeleteUserRoles implements userport.UserRolePersistencePort.
func (a *UserRoleMongoAdapter) DeleteUserRoles(
	ctx context.Context, userID uuid.UUID,
) error {
	return a.softDeleteByFilter(ctx, bson.M{
		"spec.userID":        userID.String(),
		"metadata.deletedAt": nil,
	})
}

// DeleteRoleUsers implements userport.UserRolePersistencePort.
func (a *UserRoleMongoAdapter) DeleteRoleUsers(
	ctx context.Context, roleID uuid.UUID,
) error {
	return a.softDeleteByFilter(ctx, bson.M{
		"spec.roleID":        roleID.String(),
		"metadata.deletedAt": nil,
	})
}

func (a *UserRoleMongoAdapter) findRoleIDsByUserID(
	ctx context.Context, userID uuid.UUID,
) ([]string, error) {
	filter := bson.M{
		"spec.userID":        userID.String(),
		"metadata.deletedAt": nil,
	}

	cursor, err := a.common.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find user roles by user ID: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			a.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var userRoleEntities []entity.UserRole

	err = cursor.All(ctx, &userRoleEntities)
	if err != nil {
		return nil, fmt.Errorf("decode user roles by user ID: %w", err)
	}

	roleIDs := make([]string, 0, len(userRoleEntities))
	for _, ur := range userRoleEntities {
		roleIDs = append(roleIDs, ur.Spec.RoleID)
	}

	return roleIDs, nil
}

func (a *UserRoleMongoAdapter) findRoleIDsByUserIDAndNamespace(
	ctx context.Context, userID uuid.UUID, namespace string,
) ([]string, error) {
	filter := bson.M{
		"spec.userID": userID.String(),
		"spec.namespace": bson.M{
			"$in": []string{namespace, "*"},
		},
		"metadata.deletedAt": nil,
	}

	cursor, err := a.common.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find user roles by user ID and namespace: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			a.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var userRoleEntities []entity.UserRole

	err = cursor.All(ctx, &userRoleEntities)
	if err != nil {
		return nil, fmt.Errorf("decode user roles by user ID and namespace: %w", err)
	}

	roleIDs := make([]string, 0, len(userRoleEntities))
	for _, ur := range userRoleEntities {
		roleIDs = append(roleIDs, ur.Spec.RoleID)
	}

	return roleIDs, nil
}

func (a *UserRoleMongoAdapter) findUserIDsByRoleID(
	ctx context.Context, roleID uuid.UUID,
) ([]string, error) {
	filter := bson.M{
		"spec.roleID":        roleID.String(),
		"metadata.deletedAt": nil,
	}

	cursor, err := a.common.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find user roles by role ID: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			a.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var userRoleEntities []entity.UserRole

	err = cursor.All(ctx, &userRoleEntities)
	if err != nil {
		return nil, fmt.Errorf("decode user roles by role ID: %w", err)
	}

	userIDs := make([]string, 0, len(userRoleEntities))
	for _, ur := range userRoleEntities {
		userIDs = append(userIDs, ur.Spec.UserID)
	}

	return userIDs, nil
}

func (a *UserRoleMongoAdapter) softDeleteByFilter(
	ctx context.Context, filter bson.M,
) error {
	cursor, err := a.common.collection.Find(ctx, filter)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil
		}

		return fmt.Errorf("find user roles for soft delete: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			a.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var entities []entity.UserRole

	err = cursor.All(ctx, &entities)
	if err != nil {
		return fmt.Errorf("decode user roles for soft delete: %w", err)
	}

	for i := range entities {
		domainUserRole := entities[i].ToDomain()
		domainUserRole.Delete()

		deletedEn := entity.UserRoleFromDomain(domainUserRole)

		err = a.common.put(ctx, deletedEn)
		if err != nil {
			return fmt.Errorf("soft delete user role: %w", err)
		}
	}

	return nil
}
