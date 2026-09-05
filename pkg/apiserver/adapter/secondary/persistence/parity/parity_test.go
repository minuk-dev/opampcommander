// Package parity holds cross-implementation contract tests that assert the
// MongoDB and in-memory secondary adapters back the same driven ports with
// identical semantics (put/get/list, namespace scoping, soft delete).
//
// The suite is deliberately a single generic contract run against both
// implementations of each port, so a behavioral drift between them (the class
// of bug the historical namespaced-update "phantom orphan" belonged to) fails a
// fast unit test instead of only surfacing in the Docker E2E suite. The
// in-memory backend always runs; the MongoDB backend is skipped when the
// container provider is unavailable or under `-short`.
package parity_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// Use 4.4.10 because the CI/test hardware includes arm64 (raspberry pi), which
// newer MongoDB images do not support — matching the rest of the mongo tests.
const testMongoDBImage = "mongo:4.4.10"

// contractTime is a fixed timestamp so created/deleted stamps are deterministic.
func contractTime() time.Time {
	return time.Date(2026, time.July, 24, 0, 0, 0, 0, time.UTC)
}

// backend is one concrete repository implementation of a namespaced,
// soft-deletable aggregate T, reduced to the operations the parity suite needs.
type backend[T any] struct {
	name string
	put  func(ctx context.Context, obj T) (T, error)
	get  func(ctx context.Context, ns, name string, includeDeleted bool) (T, error)
	// listNamespace returns the items visible in ns under a default (not
	// include-deleted) read, so soft-deleted records must be absent.
	listNamespace func(ctx context.Context, ns string) ([]T, error)
}

// aggregate describes how to construct and probe one domain aggregate T
// generically, plus how to build each backend for it.
type aggregate[T any] struct {
	label       string
	makeObj     func(ns, name string) T
	setMarker   func(obj T, marker string)
	getMarker   func(obj T) string
	markDeleted func(obj T)
	inmemory    func() backend[T]
	mongo       func(db *mongo.Database) backend[T]
}

func getOptions(includeDeleted bool) *model.GetOptions {
	if !includeDeleted {
		return nil
	}

	return &model.GetOptions{IncludeDeleted: true}
}

// runAggregateParity runs the shared contract against the in-memory backend
// (always) and the MongoDB backend (when the provider is healthy and not in
// short mode), so both must satisfy the same assertions.
func runAggregateParity[T any](t *testing.T, agg aggregate[T]) {
	t.Helper()

	runContract(t, agg, agg.inmemory())

	if testing.Short() {
		return
	}

	testcontainers.SkipIfProviderIsNotHealthy(t)
	runContract(t, agg, agg.mongo(startMongoDatabase(t)))
}

// startMongoDatabase spins up a throwaway MongoDB and returns a database with
// the production schema applied (indexes the repositories rely on).
func startMongoDatabase(t *testing.T) *mongo.Database {
	t.Helper()

	ctx := t.Context()

	container, err := mongoTestContainer.Run(ctx, testMongoDBImage)
	require.NoError(t, err)

	uri, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)

	t.Cleanup(func() { require.NoError(t, client.Disconnect(ctx)) })

	database := client.Database("parity_test")
	require.NoError(t, mongodb.EnsureSchema(ctx, database, false))

	return database
}

// uniqueNamespace returns a fresh namespace so subtests sharing one backend
// instance cannot see each other's records.
func uniqueNamespace(prefix string) string {
	return prefix + "-" + uuid.NewString()[:8]
}

//nolint:thelper // subtest bodies are the assertions themselves, not helpers
func runContract[T any](t *testing.T, agg aggregate[T], b backend[T]) {
	ctx := context.Background()

	t.Run(agg.label+"/"+b.name+"/put_get_roundtrip", func(t *testing.T) {
		ns := uniqueNamespace("rt")
		obj := agg.makeObj(ns, "primary")
		agg.setMarker(obj, "v1")

		_, err := b.put(ctx, obj)
		require.NoError(t, err)

		got, err := b.get(ctx, ns, "primary", false)
		require.NoError(t, err)
		assert.Equal(t, "v1", agg.getMarker(got), "stored marker must round-trip")
	})

	t.Run(agg.label+"/"+b.name+"/get_missing_is_not_found", func(t *testing.T) {
		_, err := b.get(ctx, uniqueNamespace("missing"), "nope", false)
		require.ErrorIs(t, err, model.ErrResourceNotExist)
	})

	t.Run(agg.label+"/"+b.name+"/namespace_isolation", func(t *testing.T) {
		nsA, nsB := uniqueNamespace("iso-a"), uniqueNamespace("iso-b")

		objA := agg.makeObj(nsA, "shared-name")
		agg.setMarker(objA, "from-a")
		_, err := b.put(ctx, objA)
		require.NoError(t, err)

		// A same-named record in another namespace must not bleed across.
		objB := agg.makeObj(nsB, "shared-name")
		agg.setMarker(objB, "from-b")
		_, err = b.put(ctx, objB)
		require.NoError(t, err)

		listA, err := b.listNamespace(ctx, nsA)
		require.NoError(t, err)
		require.Len(t, listA, 1)
		assert.Equal(t, "from-a", agg.getMarker(listA[0]))

		gotB, err := b.get(ctx, nsB, "shared-name", false)
		require.NoError(t, err)
		assert.Equal(t, "from-b", agg.getMarker(gotB))
	})

	t.Run(agg.label+"/"+b.name+"/get_returns_isolated_copy", func(t *testing.T) {
		ns := uniqueNamespace("copy")
		obj := agg.makeObj(ns, "primary")
		agg.setMarker(obj, "original")
		_, err := b.put(ctx, obj)
		require.NoError(t, err)

		got, err := b.get(ctx, ns, "primary", false)
		require.NoError(t, err)

		// Mutating a Get result must not leak into the store — the in-memory
		// backend must hand out deep copies to match MongoDB's fresh-decode.
		agg.setMarker(got, "mutated")

		reread, err := b.get(ctx, ns, "primary", false)
		require.NoError(t, err)
		assert.Equal(t, "original", agg.getMarker(reread),
			"mutating a Get result must not affect the stored copy")
	})

	t.Run(agg.label+"/"+b.name+"/soft_delete_hidden_unless_included", func(t *testing.T) {
		ns := uniqueNamespace("del")
		obj := agg.makeObj(ns, "primary")
		agg.setMarker(obj, "live")
		stored, err := b.put(ctx, obj)
		require.NoError(t, err)

		agg.markDeleted(stored)
		_, err = b.put(ctx, stored)
		require.NoError(t, err)

		// Default read hides the soft-deleted record.
		_, err = b.get(ctx, ns, "primary", false)
		require.ErrorIs(t, err, model.ErrResourceNotExist)

		// IncludeDeleted surfaces it again.
		revived, err := b.get(ctx, ns, "primary", true)
		require.NoError(t, err)
		assert.Equal(t, "live", agg.getMarker(revived))

		// It is excluded from the default listing.
		list, err := b.listNamespace(ctx, ns)
		require.NoError(t, err)
		assert.Empty(t, list)
	})
}

// --- Aggregate descriptors -------------------------------------------------

func TestParity_Endpoint(t *testing.T) {
	t.Parallel()
	runAggregateParity(t, endpointAggregate())
}

func TestParity_RemoteConfigSchema(t *testing.T) {
	t.Parallel()
	runAggregateParity(t, remoteConfigSchemaAggregate())
}

func remoteConfigSchemaAggregate() aggregate[*agentmodel.RemoteConfigSchema] {
	return aggregate[*agentmodel.RemoteConfigSchema]{
		label: "remoteconfigschema",
		makeObj: func(ns, name string) *agentmodel.RemoteConfigSchema {
			return agentmodel.NewRemoteConfigSchema(ns, name, nil, contractTime(), "tester")
		},
		setMarker:   func(s *agentmodel.RemoteConfigSchema, m string) { s.Spec.Binary = m },
		getMarker:   func(s *agentmodel.RemoteConfigSchema) string { return s.Spec.Binary },
		markDeleted: func(s *agentmodel.RemoteConfigSchema) { s.MarkDeleted(contractTime().Add(time.Hour), "tester") },
		inmemory: func() backend[*agentmodel.RemoteConfigSchema] {
			repo := inmemory.NewRemoteConfigSchemaRepository()

			return backend[*agentmodel.RemoteConfigSchema]{
				name: "inmemory",
				put:  repo.PutRemoteConfigSchema,
				get: func(ctx context.Context, ns, name string, incl bool) (*agentmodel.RemoteConfigSchema, error) {
					return repo.GetRemoteConfigSchema(ctx, ns, name, getOptions(incl))
				},
				listNamespace: func(ctx context.Context, ns string) ([]*agentmodel.RemoteConfigSchema, error) {
					resp, err := repo.ListRemoteConfigSchemas(ctx, ns, nil)
					if err != nil {
						return nil, fmt.Errorf("parity list: %w", err)
					}

					return resp.Items, nil
				},
			}
		},
		mongo: func(db *mongo.Database) backend[*agentmodel.RemoteConfigSchema] {
			repo := mongodb.NewRemoteConfigSchemaRepository(db, slog.Default())

			return backend[*agentmodel.RemoteConfigSchema]{
				name: "mongodb",
				put:  repo.PutRemoteConfigSchema,
				get: func(ctx context.Context, ns, name string, incl bool) (*agentmodel.RemoteConfigSchema, error) {
					return repo.GetRemoteConfigSchema(ctx, ns, name, getOptions(incl))
				},
				listNamespace: func(ctx context.Context, ns string) ([]*agentmodel.RemoteConfigSchema, error) {
					resp, err := repo.ListRemoteConfigSchemas(ctx, ns, nil)
					if err != nil {
						return nil, fmt.Errorf("parity list: %w", err)
					}

					return resp.Items, nil
				},
			}
		},
	}
}

func endpointAggregate() aggregate[*agentmodel.Endpoint] {
	return aggregate[*agentmodel.Endpoint]{
		label: "endpoint",
		makeObj: func(ns, name string) *agentmodel.Endpoint {
			return agentmodel.NewEndpoint(ns, name, nil, contractTime(), "tester")
		},
		setMarker:   func(e *agentmodel.Endpoint, m string) { e.Spec.URL = m },
		getMarker:   func(e *agentmodel.Endpoint) string { return e.Spec.URL },
		markDeleted: func(e *agentmodel.Endpoint) { e.MarkDeleted(contractTime().Add(time.Hour), "tester") },
		inmemory: func() backend[*agentmodel.Endpoint] {
			repo := inmemory.NewEndpointRepository()

			return backend[*agentmodel.Endpoint]{
				name: "inmemory",
				put:  repo.PutEndpoint,
				get: func(ctx context.Context, ns, name string, incl bool) (*agentmodel.Endpoint, error) {
					return repo.GetEndpoint(ctx, ns, name, getOptions(incl))
				},
				listNamespace: func(ctx context.Context, ns string) ([]*agentmodel.Endpoint, error) {
					resp, err := repo.ListEndpoints(ctx, ns, nil)
					if err != nil {
						return nil, fmt.Errorf("parity list: %w", err)
					}

					return resp.Items, nil
				},
			}
		},
		mongo: func(db *mongo.Database) backend[*agentmodel.Endpoint] {
			repo := mongodb.NewEndpointRepository(db, slog.Default())

			return backend[*agentmodel.Endpoint]{
				name: "mongodb",
				put:  repo.PutEndpoint,
				get: func(ctx context.Context, ns, name string, incl bool) (*agentmodel.Endpoint, error) {
					return repo.GetEndpoint(ctx, ns, name, getOptions(incl))
				},
				listNamespace: func(ctx context.Context, ns string) ([]*agentmodel.Endpoint, error) {
					resp, err := repo.ListEndpoints(ctx, ns, nil)
					if err != nil {
						return nil, fmt.Errorf("parity list: %w", err)
					}

					return resp.Items, nil
				},
			}
		},
	}
}

func TestParity_AgentRemoteConfig(t *testing.T) {
	t.Parallel()
	runAggregateParity(t, agentRemoteConfigAggregate())
}

func agentRemoteConfigAggregate() aggregate[*agentmodel.AgentRemoteConfig] {
	// ListAgentRemoteConfigs is not namespace-scoped, so the list closures
	// filter client-side to keep the shared contract's namespace assertions
	// uniform across aggregates.
	filterNS := func(items []*agentmodel.AgentRemoteConfig, ns string) []*agentmodel.AgentRemoteConfig {
		out := make([]*agentmodel.AgentRemoteConfig, 0, len(items))
		for _, it := range items {
			if it.Metadata.Namespace == ns {
				out = append(out, it)
			}
		}

		return out
	}

	return aggregate[*agentmodel.AgentRemoteConfig]{
		label: "agentremoteconfig",
		makeObj: func(ns, name string) *agentmodel.AgentRemoteConfig {
			return &agentmodel.AgentRemoteConfig{
				Metadata: agentmodel.AgentRemoteConfigMetadata{Name: name, Namespace: ns},
				Spec:     agentmodel.AgentRemoteConfigSpec{Value: []byte("body"), ContentType: "text/yaml"},
				Status:   agentmodel.AgentRemoteConfigResourceStatus{},
			}
		},
		setMarker:   func(c *agentmodel.AgentRemoteConfig, m string) { c.Spec.ContentType = m },
		getMarker:   func(c *agentmodel.AgentRemoteConfig) string { return c.Spec.ContentType },
		markDeleted: func(c *agentmodel.AgentRemoteConfig) { c.MarkDeleted(contractTime().Add(time.Hour), "tester") },
		inmemory: func() backend[*agentmodel.AgentRemoteConfig] {
			repo := inmemory.NewAgentRemoteConfigRepository()

			return backend[*agentmodel.AgentRemoteConfig]{
				name: "inmemory",
				put:  repo.PutAgentRemoteConfig,
				get: func(ctx context.Context, ns, name string, incl bool) (*agentmodel.AgentRemoteConfig, error) {
					return repo.GetAgentRemoteConfig(ctx, ns, name, getOptions(incl))
				},
				listNamespace: func(ctx context.Context, ns string) ([]*agentmodel.AgentRemoteConfig, error) {
					resp, err := repo.ListAgentRemoteConfigs(ctx, "", nil)
					if err != nil {
						return nil, fmt.Errorf("parity list: %w", err)
					}

					return filterNS(resp.Items, ns), nil
				},
			}
		},
		mongo: func(db *mongo.Database) backend[*agentmodel.AgentRemoteConfig] {
			repo := mongodb.NewAgentRemoteConfigRepository(db, slog.Default())

			return backend[*agentmodel.AgentRemoteConfig]{
				name: "mongodb",
				put:  repo.PutAgentRemoteConfig,
				get: func(ctx context.Context, ns, name string, incl bool) (*agentmodel.AgentRemoteConfig, error) {
					return repo.GetAgentRemoteConfig(ctx, ns, name, getOptions(incl))
				},
				listNamespace: func(ctx context.Context, ns string) ([]*agentmodel.AgentRemoteConfig, error) {
					resp, err := repo.ListAgentRemoteConfigs(ctx, "", nil)
					if err != nil {
						return nil, fmt.Errorf("parity list: %w", err)
					}

					return filterNS(resp.Items, ns), nil
				},
			}
		},
	}
}

func TestParity_Certificate(t *testing.T) {
	t.Parallel()
	runAggregateParity(t, certificateAggregate())
}

func certificateAggregate() aggregate[*agentmodel.Certificate] {
	filterNS := func(items []*agentmodel.Certificate, ns string) []*agentmodel.Certificate {
		out := make([]*agentmodel.Certificate, 0, len(items))
		for _, it := range items {
			if it.Metadata.Namespace == ns {
				out = append(out, it)
			}
		}

		return out
	}

	return aggregate[*agentmodel.Certificate]{
		label: "certificate",
		makeObj: func(ns, name string) *agentmodel.Certificate {
			return &agentmodel.Certificate{
				Metadata: agentmodel.CertificateMetadata{Name: name, Namespace: ns},
				Spec:     agentmodel.CertificateSpec{},
				Status:   agentmodel.CertificateStatus{},
			}
		},
		setMarker:   func(c *agentmodel.Certificate, m string) { c.Spec.Cert = []byte(m) },
		getMarker:   func(c *agentmodel.Certificate) string { return string(c.Spec.Cert) },
		markDeleted: func(c *agentmodel.Certificate) { c.MarkAsDeleted(contractTime().Add(time.Hour), "tester") },
		inmemory: func() backend[*agentmodel.Certificate] {
			repo := inmemory.NewCertificateRepository()

			return backend[*agentmodel.Certificate]{
				name: "inmemory",
				put:  repo.PutCertificate,
				get: func(ctx context.Context, ns, name string, incl bool) (*agentmodel.Certificate, error) {
					return repo.GetCertificate(ctx, ns, name, getOptions(incl))
				},
				listNamespace: func(ctx context.Context, ns string) ([]*agentmodel.Certificate, error) {
					resp, err := repo.ListCertificate(ctx, "", nil)
					if err != nil {
						return nil, fmt.Errorf("parity list: %w", err)
					}

					return filterNS(resp.Items, ns), nil
				},
			}
		},
		mongo: func(db *mongo.Database) backend[*agentmodel.Certificate] {
			repo := mongodb.NewCertificateRepository(db, slog.Default())

			return backend[*agentmodel.Certificate]{
				name: "mongodb",
				put:  repo.PutCertificate,
				get: func(ctx context.Context, ns, name string, incl bool) (*agentmodel.Certificate, error) {
					return repo.GetCertificate(ctx, ns, name, getOptions(incl))
				},
				listNamespace: func(ctx context.Context, ns string) ([]*agentmodel.Certificate, error) {
					resp, err := repo.ListCertificate(ctx, "", nil)
					if err != nil {
						return nil, fmt.Errorf("parity list: %w", err)
					}

					return filterNS(resp.Items, ns), nil
				},
			}
		},
	}
}
