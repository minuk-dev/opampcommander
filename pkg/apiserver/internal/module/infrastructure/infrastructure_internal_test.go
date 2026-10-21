package infrastructure

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
)

func TestInMemoryCasbinEnforcer(t *testing.T) {
	t.Parallel()

	enforcer, err := provideInMemoryCasbinEnforcer(slog.Default())
	require.NoError(t, err)

	ctx := t.Context()
	added, err := enforcer.AddGroupingPolicy(ctx, "alice", "reader", "default")
	require.NoError(t, err)
	require.True(t, added)
	added, err = enforcer.AddNamedPolicy(ctx, "p", "reader", "default", "agent", "GET")
	require.NoError(t, err)
	require.True(t, added)
	require.NoError(t, enforcer.BuildRoleLinks(ctx))
	require.NoError(t, enforcer.SavePolicy(ctx))

	grouping, err := enforcer.GetGroupingPolicy()
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"alice", "reader", "default"}}, grouping)

	policies, err := enforcer.GetNamedPolicy("p")
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"reader", "default", "agent", "GET"}}, policies)

	assignments, err := enforcer.GetRoleAssignmentsForUser(ctx, "alice")
	require.NoError(t, err)
	assert.Equal(t, []userport.UserRoleAssignment{{RoleUID: "reader", Namespace: "default"}}, assignments)

	allowed, err := enforcer.CheckPermission(ctx, "alice", "default", "agent", "GET")
	require.NoError(t, err)
	assert.True(t, allowed)
	allowed, err = enforcer.CheckPermission(ctx, "alice", "other", "agent", "GET")
	require.NoError(t, err)
	assert.False(t, allowed)
	allowed, err = enforcer.CheckPermission(ctx, "alice", "default", "agent", "DELETE")
	require.NoError(t, err)
	assert.False(t, allowed)
}
