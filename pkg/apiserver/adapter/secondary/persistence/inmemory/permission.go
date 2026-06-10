package inmemory

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
)

var _ userport.PermissionPersistencePort = (*PermissionRepository)(nil)

// PermissionRepository is the in-memory implementation of
// [userport.PermissionPersistencePort].
type PermissionRepository struct {
	store *store[uuid.UUID, *usermodel.Permission]
}

// NewPermissionRepository creates a new in-memory PermissionRepository.
func NewPermissionRepository() *PermissionRepository {
	return &PermissionRepository{
		store: newStore[uuid.UUID](func(permission *usermodel.Permission) *time.Time {
			return permission.Metadata.DeletedAt
		}),
	}
}

// GetPermission implements userport.PermissionPersistencePort.
func (r *PermissionRepository) GetPermission(_ context.Context, uid uuid.UUID) (*usermodel.Permission, error) {
	return r.store.get(uid, nil)
}

// GetPermissionByName implements userport.PermissionPersistencePort.
func (r *PermissionRepository) GetPermissionByName(_ context.Context, name string) (*usermodel.Permission, error) {
	permissions := r.store.snapshot(false, func(permission *usermodel.Permission) bool {
		return permission.Spec.Name == name
	})
	if len(permissions) == 0 {
		return nil, errResourceNotExist()
	}

	return permissions[0], nil
}

// PutPermission implements userport.PermissionPersistencePort.
func (r *PermissionRepository) PutPermission(
	_ context.Context, permission *usermodel.Permission,
) (*usermodel.Permission, error) {
	r.store.put(permission.Metadata.UID, permission)

	return permission, nil
}

// ListPermissions implements userport.PermissionPersistencePort.
func (r *PermissionRepository) ListPermissions(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*usermodel.Permission], error) {
	return r.store.list(options, nil)
}

// DeletePermission implements userport.PermissionPersistencePort. Permissions are soft-deleted.
func (r *PermissionRepository) DeletePermission(_ context.Context, uid uuid.UUID) error {
	permission, err := r.store.get(uid, nil)
	if err != nil {
		return err
	}

	permission.Delete()
	r.store.put(uid, permission)

	return nil
}
