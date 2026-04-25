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
	"github.com/minuk-dev/opampcommander/pkg/client"
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

// newPermissionSeed creates a permission seed entry.
func newPermissionSeed(resource, action string) permissionSeed {
	return permissionSeed{
		uid:         uuid.New().String(),
		name:        resource + ":" + action,
		description: fmt.Sprintf("%s %s permission", action, resource),
		resource:    resource,
		action:      action,
	}
}

// allPermissionSeeds returns permission seeds for all namespace-scoped
// resources and actions.
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

	// --- Infrastructure setup ---
	mongoContainer, mongoURI := startMongoDB(t)
	defer func() { _ = mongoContainer.Terminate(ctx) }()

	const dbName = "opampcommander_e2e_rbac_ns_test"

	apiPort := base.GetFreeTCPPort()

	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, dbName)
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	opampClient := createOpampClient(t, apiBaseURL)

	// --- Seed permissions in MongoDB ---
	permissions := allPermissionSeeds()
	seedPermissions(t, ctx, mongoURI, dbName, permissions)

	// Build a lookup: "resource:action" → permission name.
	permName := func(resource, action string) string {
		return resource + ":" + action
	}

	// --- Create a non-admin user (for RBAC testing) ---
	regularUser := createUser(t, opampClient, "rbac-user", "rbac-user@example.com")
	regularUserUID := regularUser.Metadata.UID

	// =================================================================
	// Phase 1: Define Roles (like K8s Role)
	// =================================================================
	// Namespace: "production", "staging", "development"
	_ = createNamespace(t, opampClient, "production")

	// Role: agent-viewer — read-only access to agents.
	_ = createRole(t, opampClient, &v1.Role{
		Kind:       v1.RoleKind,
		APIVersion: "v1",
		//exhaustruct:ignore
		Metadata: v1.RoleMetadata{},
		Spec: v1.RoleSpec{
			DisplayName: "Agent Viewer",
			Description: "Read-only access to agents",
			Permissions: []string{
				permName("agent", "GET"),
				permName("agent", "LIST"),
			},
			IsBuiltIn: false,
		},
		//exhaustruct:ignore
		Status: v1.RoleStatus{},
	})

	// Role: agent-editor — full CRUD on agents.
	_ = createRole(t, opampClient, &v1.Role{
		Kind:       v1.RoleKind,
		APIVersion: "v1",
		//exhaustruct:ignore
		Metadata: v1.RoleMetadata{},
		Spec: v1.RoleSpec{
			DisplayName: "Agent Editor",
			Description: "Full CRUD access to agents",
			Permissions: []string{
				permName("agent", "GET"),
				permName("agent", "LIST"),
				permName("agent", "CREATE"),
				permName("agent", "UPDATE"),
				permName("agent", "DELETE"),
			},
			IsBuiltIn: false,
		},
		//exhaustruct:ignore
		Status: v1.RoleStatus{},
	})

	// Role: agentgroup-admin — full CRUD on agentgroups (cluster-wide).
	_ = createRole(t, opampClient, &v1.Role{
		Kind:       v1.RoleKind,
		APIVersion: "v1",
		//exhaustruct:ignore
		Metadata: v1.RoleMetadata{},
		Spec: v1.RoleSpec{
			DisplayName: "AgentGroup Admin",
			Description: "Full CRUD access to agent groups across all namespaces",
			Permissions: []string{
				permName("agentgroup", "GET"),
				permName("agentgroup", "LIST"),
				permName("agentgroup", "CREATE"),
				permName("agentgroup", "UPDATE"),
				permName("agentgroup", "DELETE"),
			},
			IsBuiltIn: false,
		},
		//exhaustruct:ignore
		Status: v1.RoleStatus{},
	})

	// =================================================================
	// Phase 2: RoleBinding — bind agent-viewer to "production" namespace
	// (like K8s RoleBinding)
	// =================================================================
	t.Run("Phase2_BindViewerToProduction", func(t *testing.T) {
		createRoleBinding(t, opampClient, "production", "viewer-production",
			"Agent Viewer", "rbac-user@example.com")

		syncPolicies(t, opampClient)

		// ✅ Can GET and LIST agents in production.
		assertPermission(t, opampClient, regularUserUID,
			"production", "agent", "GET", true)
		assertPermission(t, opampClient, regularUserUID,
			"production", "agent", "LIST", true)

		// ❌ Cannot CREATE/UPDATE/DELETE agents in production (viewer only).
		assertPermission(t, opampClient, regularUserUID,
			"production", "agent", "CREATE", false)
		assertPermission(t, opampClient, regularUserUID,
			"production", "agent", "UPDATE", false)
		assertPermission(t, opampClient, regularUserUID,
			"production", "agent", "DELETE", false)

		// ❌ Cannot access agents in a different namespace.
		assertPermission(t, opampClient, regularUserUID,
			"staging", "agent", "GET", false)
		assertPermission(t, opampClient, regularUserUID,
			"staging", "agent", "LIST", false)

		// ❌ No agentgroup permissions at all.
		assertPermission(t, opampClient, regularUserUID,
			"production", "agentgroup", "GET", false)
	})

	// =================================================================
	// Phase 3: Additional RoleBinding — bind agent-editor to "staging"
	// =================================================================
	t.Run("Phase3_BindEditorToStaging", func(t *testing.T) {
		createRoleBinding(t, opampClient, "staging", "editor-staging",
			"Agent Editor", "rbac-user@example.com")

		syncPolicies(t, opampClient)

		// ✅ Full CRUD on agents in staging (editor role).
		assertPermission(t, opampClient, regularUserUID,
			"staging", "agent", "GET", true)
		assertPermission(t, opampClient, regularUserUID,
			"staging", "agent", "CREATE", true)
		assertPermission(t, opampClient, regularUserUID,
			"staging", "agent", "UPDATE", true)
		assertPermission(t, opampClient, regularUserUID,
			"staging", "agent", "DELETE", true)

		// ✅ Previous binding still works — viewer in production.
		assertPermission(t, opampClient, regularUserUID,
			"production", "agent", "GET", true)

		// ❌ Still cannot write in production (only viewer there).
		assertPermission(t, opampClient, regularUserUID,
			"production", "agent", "CREATE", false)

		// ❌ No access to a namespace without binding.
		assertPermission(t, opampClient, regularUserUID,
			"development", "agent", "GET", false)
	})

	// =================================================================
	// Phase 4: Unassign — remove viewer from production
	// =================================================================
	t.Run("Phase5_UnassignViewerFromProduction", func(t *testing.T) {
		deleteRoleBinding(t, opampClient, "production", "viewer-production")

		syncPolicies(t, opampClient)

		// ❌ No longer can access agents in production.
		assertPermission(t, opampClient, regularUserUID,
			"production", "agent", "GET", false)
		assertPermission(t, opampClient, regularUserUID,
			"production", "agent", "LIST", false)

		// ✅ Other bindings remain: editor in staging.
		assertPermission(t, opampClient, regularUserUID,
			"staging", "agent", "GET", true)

		// ✅ Cluster-wide agentgroup admin still active.
		assertPermission(t, opampClient, regularUserUID,
			"production", "agentgroup", "GET", true)
	})

	// =================================================================
	// Phase 5: Multi-user isolation
	// =================================================================
	t.Run("Phase6_MultiUserIsolation", func(t *testing.T) {
		// Create another user.
		otherUser := createUser(t, opampClient,
			"other-user", "other@example.com")
		otherUserUID := otherUser.Metadata.UID

		// Bind agent-viewer to other user in "staging" only.
		createRoleBinding(t, opampClient, "staging", "viewer-staging-other",
			"Agent Viewer", "other@example.com")

		syncPolicies(t, opampClient)

		// ✅ other-user can view agents in staging.
		assertPermission(t, opampClient, otherUserUID,
			"staging", "agent", "GET", true)

		// ❌ other-user CANNOT write agents in staging.
		assertPermission(t, opampClient, otherUserUID,
			"staging", "agent", "CREATE", false)

		// ❌ other-user has NO access to production agents.
		assertPermission(t, opampClient, otherUserUID,
			"production", "agent", "GET", false)

		// ❌ other-user has NO agentgroup access (not bound).
		assertPermission(t, opampClient, otherUserUID,
			"staging", "agentgroup", "GET", false)

		// ✅ original regularUser still retains their bindings.
		assertPermission(t, opampClient, regularUserUID,
			"staging", "agent", "UPDATE", true)
		assertPermission(t, opampClient, regularUserUID,
			"staging", "agentgroup", "DELETE", true)
	})
}

// --------------- Helper Functions ---------------

// createNamespace creates a new namespace via the API.
func createNamespace(t *testing.T, c *client.Client, name string) v1.Namespace {
	t.Helper()

	result, err := c.NamespaceService.CreateNamespace(t.Context(), &v1.Namespace{
		Kind:       v1.NamespaceKind,
		APIVersion: "v1",
		//exhaustruct:ignore
		Metadata: v1.NamespaceMetadata{
			Name: name,
		},
	})
	require.NoError(t, err, "failed to create namespace %s", name)

	return *result
}

// createUser creates a new user via the API and returns the user object.
func createUser(t *testing.T, c *client.Client, username, email string) v1.User {
	t.Helper()

	result, err := c.UserService.CreateUser(t.Context(), &v1.User{
		Kind:       v1.UserKind,
		APIVersion: "v1",
		//exhaustruct:ignore
		Metadata: v1.UserMetadata{},
		Spec: v1.UserSpec{
			Email:    email,
			Username: username,
			IsActive: true,
		},
		//exhaustruct:ignore
		Status: v1.UserStatus{},
	})
	require.NoError(t, err, "failed to create user %s", username)

	return *result
}

// syncPolicies triggers a full RBAC policy sync.
func syncPolicies(t *testing.T, c *client.Client) {
	t.Helper()

	err := c.RBACService.SyncPolicies(t.Context())
	require.NoError(t, err, "failed to sync RBAC policies")
}

// assertPermission checks whether a user has (or lacks) a specific
// permission and fails the test if the result doesn't match expected.
func assertPermission(
	t *testing.T,
	c *client.Client,
	userID, namespace, resource, action string,
	expected bool,
) {
	t.Helper()

	result, err := c.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
		UserID:    userID,
		Namespace: namespace,
		Resource:  resource,
		Action:    action,
	})
	require.NoError(t, err)

	if expected {
		assert.True(t, result.Allowed,
			"expected ALLOWED: user=%s namespace=%s resource=%s action=%s",
			userID, namespace, resource, action)
	} else {
		assert.False(t, result.Allowed,
			"expected DENIED: user=%s namespace=%s resource=%s action=%s",
			userID, namespace, resource, action)
	}
}
