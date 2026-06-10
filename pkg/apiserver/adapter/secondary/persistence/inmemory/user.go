package inmemory

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
)

var _ userport.UserPersistencePort = (*UserRepository)(nil)

// UserRepository is the in-memory implementation of [userport.UserPersistencePort].
type UserRepository struct {
	store *store[uuid.UUID, *usermodel.User]
}

// NewUserRepository creates a new in-memory UserRepository.
func NewUserRepository() *UserRepository {
	return &UserRepository{
		store: newStore[uuid.UUID](func(user *usermodel.User) *time.Time {
			return user.Metadata.DeletedAt
		}),
	}
}

// GetUser implements userport.UserPersistencePort.
func (r *UserRepository) GetUser(
	_ context.Context, uid uuid.UUID, options *model.GetOptions,
) (*usermodel.User, error) {
	return r.store.get(uid, options)
}

// GetUserByEmail implements userport.UserPersistencePort.
func (r *UserRepository) GetUserByEmail(_ context.Context, email string) (*usermodel.User, error) {
	return r.findByEmail(email, false)
}

// GetUserByEmailIncludingDeleted implements userport.UserPersistencePort.
func (r *UserRepository) GetUserByEmailIncludingDeleted(_ context.Context, email string) (*usermodel.User, error) {
	return r.findByEmail(email, true)
}

// PutUser implements userport.UserPersistencePort.
func (r *UserRepository) PutUser(_ context.Context, user *usermodel.User) (*usermodel.User, error) {
	r.store.put(user.Metadata.UID, user)

	return user, nil
}

// ListUsers implements userport.UserPersistencePort.
func (r *UserRepository) ListUsers(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*usermodel.User], error) {
	return r.store.list(options, nil)
}

// DeleteUser implements userport.UserPersistencePort. Users are soft-deleted.
func (r *UserRepository) DeleteUser(_ context.Context, uid uuid.UUID) error {
	user, err := r.store.get(uid, nil)
	if err != nil {
		return err
	}

	user.Delete()
	r.store.put(uid, user)

	return nil
}

func (r *UserRepository) findByEmail(email string, includeDeleted bool) (*usermodel.User, error) {
	users := r.store.snapshot(includeDeleted, func(user *usermodel.User) bool {
		return user.Spec.Email == email
	})
	if len(users) == 0 {
		return nil, errResourceNotExist()
	}

	return users[0], nil
}
