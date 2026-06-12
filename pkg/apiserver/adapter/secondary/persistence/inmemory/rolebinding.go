package inmemory

import (
	"context"
	"time"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
)

var _ userport.RoleBindingPersistencePort = (*RoleBindingRepository)(nil)

// RoleBindingRepository is the in-memory implementation of
// [userport.RoleBindingPersistencePort].
type RoleBindingRepository struct {
	store *store[namespacedName, *usermodel.RoleBinding]
}

// NewRoleBindingRepository creates a new in-memory RoleBindingRepository.
func NewRoleBindingRepository() *RoleBindingRepository {
	return &RoleBindingRepository{
		store: newStore[namespacedName](cloneRoleBinding, func(rb *usermodel.RoleBinding) *time.Time {
			return rb.Metadata.DeletedAt
		}),
	}
}

// GetRoleBinding implements userport.RoleBindingPersistencePort.
func (r *RoleBindingRepository) GetRoleBinding(
	_ context.Context, namespace, name string, options *model.GetOptions,
) (*usermodel.RoleBinding, error) {
	return r.store.get(namespacedName{Namespace: namespace, Name: name}, options)
}

// PutRoleBinding implements userport.RoleBindingPersistencePort.
func (r *RoleBindingRepository) PutRoleBinding(
	_ context.Context, rb *usermodel.RoleBinding,
) (*usermodel.RoleBinding, error) {
	r.store.put(namespacedName{Namespace: rb.Metadata.Namespace, Name: rb.Metadata.Name}, rb)

	return rb, nil
}

// ListRoleBindings implements userport.RoleBindingPersistencePort.
func (r *RoleBindingRepository) ListRoleBindings(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*usermodel.RoleBinding], error) {
	return r.store.list(options, nil)
}

// DeleteRoleBinding implements userport.RoleBindingPersistencePort. Bindings are soft-deleted.
func (r *RoleBindingRepository) DeleteRoleBinding(_ context.Context, namespace, name string) error {
	key := namespacedName{Namespace: namespace, Name: name}

	roleBinding, err := r.store.get(key, nil)
	if err != nil {
		return err
	}

	roleBinding.MarkDeleted()
	r.store.put(key, roleBinding)

	return nil
}
