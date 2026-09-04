package parity_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
)

// Role bindings are the selector contract's other shape: a resource with no label
// map at all. That makes two things worth pinning down across both adapters —
// that a name prefix and a field selector still narrow the listing, and that a
// label selector is an error rather than an empty page, which a client would
// otherwise read as "no binding matches" instead of "bindings have no labels".
//
// The listing is also namespace-scoped, so the contract seeds a second namespace
// that must never appear in the results.

// roleBindingLister is one repository reduced to a namespaced listing that
// reports the names it returned.
type roleBindingLister func(
	ctx context.Context, namespace string, options *model.ListOptions,
) ([]string, error)

type roleBindingBackend struct {
	name string
	put  func(ctx context.Context, rb *usermodel.RoleBinding) (*usermodel.RoleBinding, error)
	list roleBindingLister
}

func TestParity_RoleBindingSelectors(t *testing.T) {
	t.Parallel()

	runRoleBindingContract(t, inmemoryRoleBindingBackend())

	if testing.Short() {
		return
	}

	testcontainers.SkipIfProviderIsNotHealthy(t)
	runRoleBindingContract(t, mongoRoleBindingBackend(startMongoDatabase(t)))
}

//nolint:thelper // subtest bodies are the assertions themselves, not helpers
func runRoleBindingContract(t *testing.T, backend roleBindingBackend) {
	ctx := context.Background()
	namespace := uniqueNamespace("rbsel")
	other := uniqueNamespace("rbother")

	seed := []struct {
		namespace string
		name      string
		role      string
	}{
		{namespace, "prod-admins", "Admin"},
		{namespace, "prod-viewers", "Viewer"},
		{namespace, "staging-viewers", "Viewer"},
		{other, "prod-admins", "Admin"},
	}

	for _, spec := range seed {
		binding := usermodel.NewRoleBinding(spec.namespace, spec.name,
			usermodel.RoleRef{Kind: usermodel.SubjectKindUser, Name: spec.role})

		_, err := backend.put(ctx, binding)
		require.NoError(t, err)
	}

	//exhaustruct:ignore
	cases := []struct {
		name          string
		fieldSelector string
		namePrefix    string
		expected      []string
	}{
		{
			name:     "no selector returns only this namespace",
			expected: []string{"prod-admins", "prod-viewers", "staging-viewers"},
		},
		{
			name:       "name prefix",
			namePrefix: "prod-",
			expected:   []string{"prod-admins", "prod-viewers"},
		},
		{
			name:          "field selector on the referenced role",
			fieldSelector: "spec.roleRef.name=Viewer",
			expected:      []string{"prod-viewers", "staging-viewers"},
		},
		{
			name:          "name prefix combined with a field selector",
			namePrefix:    "prod-",
			fieldSelector: "spec.roleRef.name=Viewer",
			expected:      []string{"prod-viewers"},
		},
		{
			name:          "inequality on the referenced role",
			fieldSelector: "spec.roleRef.name!=Viewer",
			expected:      []string{"prod-admins"},
		},
		{
			name:          "a selector matching nothing returns nothing",
			fieldSelector: "spec.roleRef.name=Nobody",
			expected:      nil,
		},
	}

	for _, testCase := range cases {
		t.Run(backend.name+"/"+testCase.name, func(t *testing.T) {
			t.Parallel()

			//exhaustruct:ignore
			options := &model.ListOptions{
				FieldSelector: parseFields(t, testCase.fieldSelector),
				NamePrefix:    testCase.namePrefix,
			}

			names, err := backend.list(ctx, namespace, options)
			require.NoError(t, err)
			assert.ElementsMatch(t, testCase.expected, names)
		})
	}

	t.Run(backend.name+"/a_label_selector_is_rejected_not_answered_empty", func(t *testing.T) {
		t.Parallel()

		//exhaustruct:ignore
		options := &model.ListOptions{LabelSelector: parseLabels(t, "env=prod")}

		_, err := backend.list(ctx, namespace, options)
		require.ErrorIs(t, err, model.ErrInvalidArgument)
	})

	t.Run(backend.name+"/every_namespace", func(t *testing.T) {
		t.Parallel()

		// The empty namespace is what RBAC policy loading passes: it needs every
		// binding, wherever it lives, to compute a user's effective permissions.
		//exhaustruct:ignore
		names, err := backend.list(ctx, "", &model.ListOptions{NamePrefix: "prod-admins"})
		require.NoError(t, err)
		assert.Len(t, names, 2, "one binding of this name in each seeded namespace")
	})
}

func roleBindingListerFor(
	list func(ctx context.Context, namespace string, options *model.ListOptions) (
		*model.ListResponse[*usermodel.RoleBinding], error),
) roleBindingLister {
	return func(ctx context.Context, namespace string, options *model.ListOptions) ([]string, error) {
		resp, err := list(ctx, namespace, options)
		if err != nil {
			return nil, fmt.Errorf("parity list: %w", err)
		}

		names := make([]string, 0, len(resp.Items))
		for _, item := range resp.Items {
			names = append(names, item.Metadata.Name)
		}

		return names, nil
	}
}

func inmemoryRoleBindingBackend() roleBindingBackend {
	repo := inmemory.NewRoleBindingRepository()

	return roleBindingBackend{
		name: "rolebinding/inmemory",
		put:  repo.PutRoleBinding,
		list: roleBindingListerFor(repo.ListRoleBindings),
	}
}

func mongoRoleBindingBackend(database *mongo.Database) roleBindingBackend {
	repo := mongodb.NewRoleBindingRepository(database, slog.Default())

	return roleBindingBackend{
		name: "rolebinding/mongodb",
		put:  repo.PutRoleBinding,
		list: roleBindingListerFor(repo.ListRoleBindings),
	}
}
