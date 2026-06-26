package userservice_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
	userservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/service"
)

// roleFakePersistence is a minimal in-memory RolePersistencePort for the role
// lifecycle tests.
type roleFakePersistence struct {
	stored   *usermodel.Role
	getErr   error
	putCalls int
	lastPut  *usermodel.Role
}

func (f *roleFakePersistence) GetRole(
	_ context.Context, _ uuid.UUID, _ *model.GetOptions,
) (*usermodel.Role, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}

	return f.stored, nil
}

func (f *roleFakePersistence) GetRoleByName(_ context.Context, _ string) (*usermodel.Role, error) {
	return f.stored, nil
}

func (f *roleFakePersistence) PutRole(_ context.Context, role *usermodel.Role) (*usermodel.Role, error) {
	f.putCalls++
	f.lastPut = role

	return role, nil
}

func (f *roleFakePersistence) ListRoles(
	_ context.Context, _ *model.ListOptions,
) (*model.ListResponse[*usermodel.Role], error) {
	return &model.ListResponse[*usermodel.Role]{}, nil
}

func (f *roleFakePersistence) DeleteRole(_ context.Context, _ uuid.UUID) error {
	return nil
}

var _ userport.RolePersistencePort = (*roleFakePersistence)(nil)

func TestRoleService_CreateRole_ForcesNotBuiltIn(t *testing.T) {
	t.Parallel()

	persistence := &roleFakePersistence{}
	svc := userservice.NewRoleService(persistence, slog.Default())

	// A caller trying to sneak in a built-in role must be overridden.
	input := usermodel.NewRole("editor", true)

	created, err := svc.CreateRole(t.Context(), input)

	require.NoError(t, err)
	assert.Equal(t, 1, persistence.putCalls)
	assert.False(t, created.Spec.IsBuiltIn, "user-created roles must never be built-in")
	assert.NotEqual(t, uuid.Nil, created.Metadata.UID, "a created role must have an identity")
}

func TestRoleService_UpdateRole_KeepsIsBuiltInImmutable(t *testing.T) {
	t.Parallel()

	stored := usermodel.NewRole("admin", true) // a built-in role
	persistence := &roleFakePersistence{stored: stored}
	svc := userservice.NewRoleService(persistence, slog.Default())

	// The update body tries to flip IsBuiltIn off and change the spec.
	incoming := usermodel.NewRole("admin-renamed", false)
	incoming.Spec.Description = "changed"
	incoming.Spec.Permissions = []string{"agent:GET"}

	updated, err := svc.UpdateRole(t.Context(), stored.Metadata.UID, incoming)

	require.NoError(t, err)
	assert.True(t, updated.Spec.IsBuiltIn, "IsBuiltIn must be immutable on update")
	assert.Equal(t, "admin-renamed", updated.Spec.DisplayName, "mutable spec fields must be applied")
	assert.Equal(t, []string{"agent:GET"}, updated.Spec.Permissions)
}
