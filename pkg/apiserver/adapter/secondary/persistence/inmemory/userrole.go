package inmemory

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
)

var _ userport.UserRolePersistencePort = (*UserRoleRepository)(nil)

// UserRoleRepository is the in-memory implementation of
// [userport.UserRolePersistencePort]. It reads the role and user stores to
// resolve the cross-entity queries (GetUserRoles, GetRoleUsers, ...).
type UserRoleRepository struct {
	store    *store[uuid.UUID, *usermodel.UserRole]
	roleRepo *RoleRepository
	userRepo *UserRepository
}

// NewUserRoleRepository creates a new in-memory UserRoleRepository.
func NewUserRoleRepository(roleRepo *RoleRepository, userRepo *UserRepository) *UserRoleRepository {
	return &UserRoleRepository{
		store: newStore[uuid.UUID](cloneUserRole, func(ur *usermodel.UserRole) *time.Time {
			return ur.Metadata.DeletedAt
		}),
		roleRepo: roleRepo,
		userRepo: userRepo,
	}
}

// GetUserRole implements userport.UserRolePersistencePort.
func (r *UserRoleRepository) GetUserRole(_ context.Context, uid uuid.UUID) (*usermodel.UserRole, error) {
	return r.store.get(uid, nil)
}

// GetUserRoleByUserAndRole implements userport.UserRolePersistencePort.
func (r *UserRoleRepository) GetUserRoleByUserAndRole(
	_ context.Context, userID, roleID uuid.UUID, namespace string,
) (*usermodel.UserRole, error) {
	matches := r.store.snapshot(false, func(ur *usermodel.UserRole) bool {
		return ur.Spec.UserID == userID &&
			ur.Spec.RoleID == roleID &&
			ur.Spec.Namespace == namespace
	})
	if len(matches) == 0 {
		return nil, errResourceNotExist()
	}

	return matches[0], nil
}

// PutUserRole implements userport.UserRolePersistencePort.
func (r *UserRoleRepository) PutUserRole(
	_ context.Context, userRole *usermodel.UserRole,
) (*usermodel.UserRole, error) {
	r.store.put(userRole.Metadata.UID, userRole)

	return userRole, nil
}

// ListUserRoles implements userport.UserRolePersistencePort.
func (r *UserRoleRepository) ListUserRoles(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*usermodel.UserRole], error) {
	return r.store.list(options, nil)
}

// GetUserRoles implements userport.UserRolePersistencePort.
func (r *UserRoleRepository) GetUserRoles(_ context.Context, userID uuid.UUID) ([]*usermodel.Role, error) {
	roleIDs := r.roleIDsMatching(func(ur *usermodel.UserRole) bool {
		return ur.Spec.UserID == userID
	})

	return r.roleRepo.getByUIDs(roleIDs), nil
}

// GetUserRolesInNamespace implements userport.UserRolePersistencePort.
// It returns roles assigned in the given namespace or cluster-wide ("*").
func (r *UserRoleRepository) GetUserRolesInNamespace(
	_ context.Context, userID uuid.UUID, namespace string,
) ([]*usermodel.Role, error) {
	roleIDs := r.roleIDsMatching(func(ur *usermodel.UserRole) bool {
		return ur.Spec.UserID == userID &&
			(ur.Spec.Namespace == namespace || ur.Spec.Namespace == usermodel.WildcardAll)
	})

	return r.roleRepo.getByUIDs(roleIDs), nil
}

// GetRoleUsers implements userport.UserRolePersistencePort.
func (r *UserRoleRepository) GetRoleUsers(_ context.Context, roleID uuid.UUID) ([]*usermodel.User, error) {
	userRoles := r.store.snapshot(false, func(ur *usermodel.UserRole) bool {
		return ur.Spec.RoleID == roleID
	})

	users := make([]*usermodel.User, 0, len(userRoles))

	for _, ur := range userRoles {
		user, err := r.userRepo.store.get(ur.Spec.UserID, nil)
		if err != nil {
			continue
		}

		users = append(users, user)
	}

	return users, nil
}

// DeleteUserRole implements userport.UserRolePersistencePort. Assignments are soft-deleted.
func (r *UserRoleRepository) DeleteUserRole(_ context.Context, uid uuid.UUID) error {
	userRole, err := r.store.get(uid, nil)
	if err != nil {
		return err
	}

	userRole.Delete()
	r.store.put(uid, userRole)

	return nil
}

// DeleteUserRoles implements userport.UserRolePersistencePort.
func (r *UserRoleRepository) DeleteUserRoles(_ context.Context, userID uuid.UUID) error {
	r.softDeleteMatching(func(ur *usermodel.UserRole) bool {
		return ur.Spec.UserID == userID
	})

	return nil
}

// DeleteRoleUsers implements userport.UserRolePersistencePort.
func (r *UserRoleRepository) DeleteRoleUsers(_ context.Context, roleID uuid.UUID) error {
	r.softDeleteMatching(func(ur *usermodel.UserRole) bool {
		return ur.Spec.RoleID == roleID
	})

	return nil
}

// roleIDsMatching returns the role UIDs of the non-deleted assignments matching predicate.
func (r *UserRoleRepository) roleIDsMatching(predicate func(*usermodel.UserRole) bool) []uuid.UUID {
	userRoles := r.store.snapshot(false, predicate)

	roleIDs := make([]uuid.UUID, 0, len(userRoles))
	for _, ur := range userRoles {
		roleIDs = append(roleIDs, ur.Spec.RoleID)
	}

	return roleIDs
}

// softDeleteMatching marks every non-deleted assignment matching predicate as deleted.
func (r *UserRoleRepository) softDeleteMatching(predicate func(*usermodel.UserRole) bool) {
	for _, ur := range r.store.snapshot(false, predicate) {
		ur.Delete()
		r.store.put(ur.Metadata.UID, ur)
	}
}
