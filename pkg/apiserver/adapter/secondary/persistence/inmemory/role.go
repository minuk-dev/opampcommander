package inmemory

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
)

var _ userport.RolePersistencePort = (*RoleRepository)(nil)

// RoleRepository is the in-memory implementation of [userport.RolePersistencePort].
type RoleRepository struct {
	store *store[uuid.UUID, *usermodel.Role]
}

// NewRoleRepository creates a new in-memory RoleRepository.
func NewRoleRepository() *RoleRepository {
	return &RoleRepository{
		store: newStore[uuid.UUID](cloneRole, func(role *usermodel.Role) *time.Time {
			return role.Metadata.DeletedAt
		}),
	}
}

// GetRole implements userport.RolePersistencePort.
func (r *RoleRepository) GetRole(
	_ context.Context, uid uuid.UUID, options *model.GetOptions,
) (*usermodel.Role, error) {
	return r.store.get(uid, options)
}

// GetRoleByName implements userport.RolePersistencePort.
func (r *RoleRepository) GetRoleByName(_ context.Context, displayName string) (*usermodel.Role, error) {
	roles := r.store.snapshot(false, func(role *usermodel.Role) bool {
		return role.Spec.DisplayName == displayName
	})
	if len(roles) == 0 {
		return nil, errResourceNotExist()
	}

	return roles[0], nil
}

// PutRole implements userport.RolePersistencePort.
func (r *RoleRepository) PutRole(_ context.Context, role *usermodel.Role) (*usermodel.Role, error) {
	r.store.put(role.Metadata.UID, role)

	return role, nil
}

// ListRoles implements userport.RolePersistencePort.
func (r *RoleRepository) ListRoles(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*usermodel.Role], error) {
	return r.store.list(options, nil)
}

// DeleteRole implements userport.RolePersistencePort. Roles are soft-deleted.
func (r *RoleRepository) DeleteRole(_ context.Context, uid uuid.UUID) error {
	role, err := r.store.get(uid, nil)
	if err != nil {
		return err
	}

	role.Delete()
	r.store.put(uid, role)

	return nil
}

// getByUIDs returns the non-deleted roles for the given UIDs, preserving store order.
func (r *RoleRepository) getByUIDs(uids []uuid.UUID) []*usermodel.Role {
	wanted := make(map[uuid.UUID]struct{}, len(uids))
	for _, uid := range uids {
		wanted[uid] = struct{}{}
	}

	return r.store.snapshot(false, func(role *usermodel.Role) bool {
		_, ok := wanted[role.Metadata.UID]

		return ok
	})
}
