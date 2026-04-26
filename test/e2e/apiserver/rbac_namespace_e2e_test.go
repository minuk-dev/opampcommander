//go:build e2e

package apiserver_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// seedPermissions inserts permission documents directly into MongoDB.
// This is needed because there is no REST API to create permissions.
func seedPermissions(
	t *testing.T,
	ctx context.Context,
	mongoURI, dbName string,
	permissions []permissionSeed,
) {
	t.Helper()

	clientOpts := options.Client().ApplyURI(mongoURI)

	client, err := mongo.Connect(clientOpts)
	require.NoError(t, err)

	defer func() { _ = client.Disconnect(ctx) }()

	collection := client.Database(dbName).Collection("permissions")

	for _, perm := range permissions {
		now := time.Now()

		doc := bson.M{
			"version": 1,
			"metadata": bson.M{
				"uid":       perm.uid,
				"createdAt": now,
				"updatedAt": now,
			},
			"spec": bson.M{
				"name":        perm.name,
				"description": perm.description,
				"resource":    perm.resource,
				"action":      perm.action,
				"isBuiltIn":   true,
			},
			"status": bson.M{
				"conditions": bson.A{},
			},
		}

		_, err := collection.InsertOne(ctx, doc)
		require.NoError(t, err, "failed to seed permission: %s", perm.name)
	}
}

type permissionSeed struct {
	uid         string
	name        string
	description string
	resource    string
	action      string
}

func newPermissionSeed(resource, action string) permissionSeed {
	return permissionSeed{
		uid:         uuid.New().String(),
		name:        resource + ":" + action,
		description: fmt.Sprintf("%s %s permission", action, resource),
		resource:    resource,
		action:      action,
	}
}

func allPermissionSeeds() []permissionSeed {
	resources := []string{
		"agent", "agentgroup", "agentpackage",
		"certificate", "agentremoteconfig",
	}
	actions := []string{"GET", "LIST", "CREATE", "UPDATE", "DELETE"}

	seeds := make([]permissionSeed, 0, len(resources)*len(actions))
	for _, r := range resources {
		for _, a := range actions {
			seeds = append(seeds, newPermissionSeed(r, a))
		}
	}

	return seeds
}

//nolint:funlen,cyclop // E2E test function with many sequential test steps.
func TestE2E_APIServer_NamespaceScopedRBAC(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	const dbName = "opampcommander_e2e_rbac_ns_test"

	mongoServer := base.StartMongoDB()
	apiServer := base.StartAPIServer(mongoServer.URI, dbName)
	defer apiServer.Stop()

	apiServer.WaitForReady()

	opampClient := apiServer.Client()

	// --- Seed permissions in MongoDB ---
	seedPermissions(t, ctx, mongoServer.URI, dbName, allPermissionSeeds())

	permName := func(resource, action string) string { return resource + ":" + action }

	// --- Create a non-admin user (for RBAC testing) ---
	regularUser, err := opampClient.UserService.CreateUser(t.Context(), &v1.User{
		Kind:       v1.UserKind,
		APIVersion: "v1",
		//exhaustruct:ignore
		Metadata: v1.UserMetadata{},
		Spec:     v1.UserSpec{Email: "rbac-user@example.com", Username: "rbac-user", IsActive: true},
		//exhaustruct:ignore
		Status: v1.UserStatus{},
	})
	require.NoError(t, err, "failed to create regular user")

	regularUserUID := regularUser.Metadata.UID

	// =================================================================
	// Phase 1: Define Roles + Namespace
	// =================================================================
	_, err = opampClient.NamespaceService.CreateNamespace(t.Context(), &v1.Namespace{
		//exhaustruct:ignore
		Metadata: v1.NamespaceMetadata{Name: "production"},
	})
	require.NoError(t, err, "failed to create namespace production")

	// Role: agent-viewer — read-only access to agents.
	_, err = opampClient.RoleService.CreateRole(t.Context(), &v1.Role{
		Kind:       v1.RoleKind,
		APIVersion: "v1",
		//exhaustruct:ignore
		Metadata: v1.RoleMetadata{},
		Spec: v1.RoleSpec{
			DisplayName: "Agent Viewer",
			Description: "Read-only access to agents",
			Permissions: []string{permName("agent", "GET"), permName("agent", "LIST")},
			IsBuiltIn:   false,
		},
		//exhaustruct:ignore
		Status: v1.RoleStatus{},
	})
	require.NoError(t, err, "failed to create Agent Viewer role")

	// Role: agent-editor — full CRUD on agents.
	_, err = opampClient.RoleService.CreateRole(t.Context(), &v1.Role{
		Kind:       v1.RoleKind,
		APIVersion: "v1",
		//exhaustruct:ignore
		Metadata: v1.RoleMetadata{},
		Spec: v1.RoleSpec{
			DisplayName: "Agent Editor",
			Description: "Full CRUD access to agents",
			Permissions: []string{
				permName("agent", "GET"), permName("agent", "LIST"),
				permName("agent", "CREATE"), permName("agent", "UPDATE"), permName("agent", "DELETE"),
			},
			IsBuiltIn: false,
		},
		//exhaustruct:ignore
		Status: v1.RoleStatus{},
	})
	require.NoError(t, err, "failed to create Agent Editor role")

	// Role: agentgroup-admin — full CRUD on agentgroups.
	_, err = opampClient.RoleService.CreateRole(t.Context(), &v1.Role{
		Kind:       v1.RoleKind,
		APIVersion: "v1",
		//exhaustruct:ignore
		Metadata: v1.RoleMetadata{},
		Spec: v1.RoleSpec{
			DisplayName: "AgentGroup Admin",
			Description: "Full CRUD access to agent groups across all namespaces",
			Permissions: []string{
				permName("agentgroup", "GET"), permName("agentgroup", "LIST"),
				permName("agentgroup", "CREATE"), permName("agentgroup", "UPDATE"), permName("agentgroup", "DELETE"),
			},
			IsBuiltIn: false,
		},
		//exhaustruct:ignore
		Status: v1.RoleStatus{},
	})
	require.NoError(t, err, "failed to create AgentGroup Admin role")

	// =================================================================
	// Phase 2: RoleBinding — bind agent-viewer to "production" namespace
	// =================================================================
	t.Run("Phase2_BindViewerToProduction", func(t *testing.T) {
		_, err := opampClient.RoleBindingService.CreateRoleBinding(t.Context(), "production", &v1.RoleBinding{
			Kind:       v1.RoleBindingKind,
			APIVersion: "v1",
			Metadata: v1.RoleBindingMetadata{
				Namespace: "production", Name: "viewer-production",
				//exhaustruct:ignore
				CreatedAt: v1.Time{},
				//exhaustruct:ignore
				UpdatedAt: v1.Time{},
			},
			Spec: v1.RoleBindingSpec{
				RoleRef: v1.RoleBindingRoleRef{Kind: "Role", Name: "Agent Viewer"},
				Subject: v1.RoleBindingSubject{Kind: "User", Name: "rbac-user@example.com"},
			},
			//exhaustruct:ignore
			Status: v1.RoleBindingStatus{},
		})
		require.NoError(t, err)

		require.NoError(t, opampClient.RBACService.SyncPolicies(t.Context()))

		// ✅ Can GET and LIST agents in production.
		perm, err := opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "production", Resource: "agent", Action: "GET",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: production/agent/GET")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "production", Resource: "agent", Action: "LIST",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: production/agent/LIST")

		// ❌ Cannot CREATE/UPDATE/DELETE agents in production (viewer only).
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "production", Resource: "agent", Action: "CREATE",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: production/agent/CREATE")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "production", Resource: "agent", Action: "UPDATE",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: production/agent/UPDATE")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "production", Resource: "agent", Action: "DELETE",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: production/agent/DELETE")

		// ❌ Cannot access agents in a different namespace.
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "staging", Resource: "agent", Action: "GET",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: staging/agent/GET")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "staging", Resource: "agent", Action: "LIST",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: staging/agent/LIST")

		// ❌ No agentgroup permissions at all.
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "production", Resource: "agentgroup", Action: "GET",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: production/agentgroup/GET")
	})

	// =================================================================
	// Phase 3: Additional RoleBinding — bind agent-editor to "staging"
	// =================================================================
	t.Run("Phase3_BindEditorToStaging", func(t *testing.T) {
		_, err := opampClient.RoleBindingService.CreateRoleBinding(t.Context(), "staging", &v1.RoleBinding{
			Kind:       v1.RoleBindingKind,
			APIVersion: "v1",
			Metadata: v1.RoleBindingMetadata{
				Namespace: "staging", Name: "editor-staging",
				//exhaustruct:ignore
				CreatedAt: v1.Time{},
				//exhaustruct:ignore
				UpdatedAt: v1.Time{},
			},
			Spec: v1.RoleBindingSpec{
				RoleRef: v1.RoleBindingRoleRef{Kind: "Role", Name: "Agent Editor"},
				Subject: v1.RoleBindingSubject{Kind: "User", Name: "rbac-user@example.com"},
			},
			//exhaustruct:ignore
			Status: v1.RoleBindingStatus{},
		})
		require.NoError(t, err)

		require.NoError(t, opampClient.RBACService.SyncPolicies(t.Context()))

		// ✅ Full CRUD on agents in staging (editor role).
		perm, err := opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "staging", Resource: "agent", Action: "GET",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: staging/agent/GET")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "staging", Resource: "agent", Action: "CREATE",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: staging/agent/CREATE")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "staging", Resource: "agent", Action: "UPDATE",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: staging/agent/UPDATE")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "staging", Resource: "agent", Action: "DELETE",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: staging/agent/DELETE")

		// ✅ Previous binding still works — viewer in production.
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "production", Resource: "agent", Action: "GET",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: production/agent/GET")

		// ❌ Still cannot write in production (only viewer there).
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "production", Resource: "agent", Action: "CREATE",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: production/agent/CREATE")

		// ❌ No access to a namespace without binding.
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "development", Resource: "agent", Action: "GET",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: development/agent/GET")
	})

	// =================================================================
	// Phase 4: Bind AgentGroup Admin in production and staging namespaces
	// =================================================================
	t.Run("Phase4_BindAgentGroupAdmin", func(t *testing.T) {
		_, err := opampClient.RoleBindingService.CreateRoleBinding(t.Context(), "production", &v1.RoleBinding{
			Kind:       v1.RoleBindingKind,
			APIVersion: "v1",
			Metadata: v1.RoleBindingMetadata{
				Namespace: "production", Name: "agentgroup-admin-production",
				//exhaustruct:ignore
				CreatedAt: v1.Time{},
				//exhaustruct:ignore
				UpdatedAt: v1.Time{},
			},
			Spec: v1.RoleBindingSpec{
				RoleRef: v1.RoleBindingRoleRef{Kind: "Role", Name: "AgentGroup Admin"},
				Subject: v1.RoleBindingSubject{Kind: "User", Name: "rbac-user@example.com"},
			},
			//exhaustruct:ignore
			Status: v1.RoleBindingStatus{},
		})
		require.NoError(t, err)

		_, err = opampClient.RoleBindingService.CreateRoleBinding(t.Context(), "staging", &v1.RoleBinding{
			Kind:       v1.RoleBindingKind,
			APIVersion: "v1",
			Metadata: v1.RoleBindingMetadata{
				Namespace: "staging", Name: "agentgroup-admin-staging",
				//exhaustruct:ignore
				CreatedAt: v1.Time{},
				//exhaustruct:ignore
				UpdatedAt: v1.Time{},
			},
			Spec: v1.RoleBindingSpec{
				RoleRef: v1.RoleBindingRoleRef{Kind: "Role", Name: "AgentGroup Admin"},
				Subject: v1.RoleBindingSubject{Kind: "User", Name: "rbac-user@example.com"},
			},
			//exhaustruct:ignore
			Status: v1.RoleBindingStatus{},
		})
		require.NoError(t, err)

		require.NoError(t, opampClient.RBACService.SyncPolicies(t.Context()))

		// ✅ Can manage agentgroups in production and staging.
		perm, err := opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "production", Resource: "agentgroup", Action: "GET",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: production/agentgroup/GET")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "staging", Resource: "agentgroup", Action: "DELETE",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: staging/agentgroup/DELETE")

		// ❌ No agentgroup access in a namespace without binding.
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "development", Resource: "agentgroup", Action: "GET",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: development/agentgroup/GET")
	})

	// =================================================================
	// Phase 5: Unassign — remove viewer from production
	// =================================================================
	t.Run("Phase5_UnassignViewerFromProduction", func(t *testing.T) {
		err := opampClient.RoleBindingService.DeleteRoleBinding(t.Context(), "production", "viewer-production")
		require.NoError(t, err)

		require.NoError(t, opampClient.RBACService.SyncPolicies(t.Context()))

		// ❌ No longer can access agents in production.
		perm, err := opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "production", Resource: "agent", Action: "GET",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: production/agent/GET")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "production", Resource: "agent", Action: "LIST",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: production/agent/LIST")

		// ✅ Other bindings remain: editor in staging.
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "staging", Resource: "agent", Action: "GET",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: staging/agent/GET")

		// ✅ AgentGroup admin binding in production still active.
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "production", Resource: "agentgroup", Action: "GET",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: production/agentgroup/GET")
	})

	// =================================================================
	// Phase 6: Multi-user isolation
	// =================================================================
	t.Run("Phase6_MultiUserIsolation", func(t *testing.T) {
		otherUser, err := opampClient.UserService.CreateUser(t.Context(), &v1.User{
			Kind:       v1.UserKind,
			APIVersion: "v1",
			//exhaustruct:ignore
			Metadata: v1.UserMetadata{},
			Spec:     v1.UserSpec{Email: "other@example.com", Username: "other-user", IsActive: true},
			//exhaustruct:ignore
			Status: v1.UserStatus{},
		})
		require.NoError(t, err, "failed to create other-user")

		otherUserUID := otherUser.Metadata.UID

		_, err = opampClient.RoleBindingService.CreateRoleBinding(t.Context(), "staging", &v1.RoleBinding{
			Kind:       v1.RoleBindingKind,
			APIVersion: "v1",
			Metadata: v1.RoleBindingMetadata{
				Namespace: "staging", Name: "viewer-staging-other",
				//exhaustruct:ignore
				CreatedAt: v1.Time{},
				//exhaustruct:ignore
				UpdatedAt: v1.Time{},
			},
			Spec: v1.RoleBindingSpec{
				RoleRef: v1.RoleBindingRoleRef{Kind: "Role", Name: "Agent Viewer"},
				Subject: v1.RoleBindingSubject{Kind: "User", Name: "other@example.com"},
			},
			//exhaustruct:ignore
			Status: v1.RoleBindingStatus{},
		})
		require.NoError(t, err)

		require.NoError(t, opampClient.RBACService.SyncPolicies(t.Context()))

		// ✅ other-user can view agents in staging.
		perm, err := opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: otherUserUID, Namespace: "staging", Resource: "agent", Action: "GET",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: other/staging/agent/GET")

		// ❌ other-user CANNOT write agents in staging.
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: otherUserUID, Namespace: "staging", Resource: "agent", Action: "CREATE",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: other/staging/agent/CREATE")

		// ❌ other-user has NO access to production agents.
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: otherUserUID, Namespace: "production", Resource: "agent", Action: "GET",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: other/production/agent/GET")

		// ❌ other-user has NO agentgroup access (not bound).
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: otherUserUID, Namespace: "staging", Resource: "agentgroup", Action: "GET",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: other/staging/agentgroup/GET")

		// ✅ original regularUser still retains their bindings.
		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "staging", Resource: "agent", Action: "UPDATE",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: regular/staging/agent/UPDATE")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: regularUserUID, Namespace: "staging", Resource: "agentgroup", Action: "DELETE",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: regular/staging/agentgroup/DELETE")
	})
}
