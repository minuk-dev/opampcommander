// Package mongodb provides the MongoDB adapter for the opampcommander application.
package mongodb

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// serverHeartbeatTTL is how long a server heartbeat survives without a refresh before MongoDB's
// TTL monitor drops it. It is pure garbage collection of crashed servers, NOT the liveness
// cutoff (reads use the caller's notBefore); it must therefore stay far above any configured
// read-staleness window, or a still-"live" server's heartbeat could be reaped early. Sized at
// 24h so no realistic staleness setting approaches it.
const serverHeartbeatTTL = 24 * time.Hour

// sanitizeResourceName validates and returns a safe resource name for MongoDB queries.
// Each rune is checked against a whitelist and copied to a new string, preventing
// NoSQL injection by ensuring the output cannot contain MongoDB operators.
func sanitizeResourceName(name string) string {
	var builder strings.Builder

	builder.Grow(len(name))

	for _, r := range name {
		if isAllowedResourceNameRune(r) {
			builder.WriteRune(r)
		} else {
			return ""
		}
	}

	return builder.String()
}

func isAllowedResourceNameRune(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '.' || r == '-' || r == '_'
}

//nolint:gochecknoglobals // These are constants for collection names and indexes.
var (
	// collections is the full managed collection set. Every collection is created
	// explicitly here — rather than lazily on first insert — so that in a sharded
	// cluster each one lands deterministically (config-scale collections stay on the
	// primary shard; the sharded ones are handled by shardedCollections below).
	collections = []string{
		agentCollectionName,
		agentGroupCollectionName,
		agentPackageCollectionName,
		agentRemoteConfigCollectionName,
		certificateCollectionName,
		endpointCollectionName,
		hostCollectionName,
		containerCollectionName,
		namespaceCollectionName,
		remoteConfigSchemaCollectionName,
		serverCollectionName,
		serverConnectionCollectionName,
		serverHeartbeatCollectionName,
		userCollectionName,
		roleCollectionName,
		roleBindingCollectionName,
		permissionCollectionName,
		userRoleCollectionName,
	}

	// shardedCollections is the built-in shard-key plan. Only collections that grow
	// with fleet size are sharded; config-/cluster-scale collections stay on the
	// primary shard. Each shard key uses a hashed index for even write distribution,
	// and — per MongoDB's rule that a unique index must be prefixed by the shard key —
	// is a prefix of that collection's unique logical-key index, so hashed routing keeps
	// uniqueness single-shard.
	shardedCollections = []shardCollectionSpec{
		{
			// Largest collection; unique index is on metadata.instanceUid.
			collectionName: agentCollectionName,
			shardKey:       bson.D{{Key: "metadata.instanceUid", Value: "hashed"}},
		},
		{
			// Grows with live agent connections; uid is unique by construction.
			collectionName: serverConnectionCollectionName,
			shardKey:       bson.D{{Key: "uid", Value: "hashed"}},
		},
	}
)

// managedIndexes returns the index plan for every managed collection.
//
// It is built fresh on each call rather than held in a package-level variable:
// the driver mutates the option builders it is handed (CreateMany names any index
// that does not carry one), so a shared plan is a data race between two
// concurrent EnsureSchema calls — which is exactly what a test run that spins up
// several databases at once does.
func managedIndexes() []collectionAndIndexes {
	return slices.Concat(fleetIndexes(), identityIndexes())
}

// fleetIndexes is the index plan for the collections that describe the managed
// fleet: agents and everything configured for them, plus the server bookkeeping
// that tracks where they are connected.
//
//nolint:funlen // one declarative table; splitting it further would scatter the schema
func fleetIndexes() []collectionAndIndexes {
	return []collectionAndIndexes{
		{
			collectionName: agentCollectionName,
			indexes: []mongo.IndexModel{
				// Unique on the logical key. Besides preventing duplicate agent
				// documents, this is what makes PutAgent's optimistic-concurrency
				// create path correct: a racing insert for an existing instanceUid is
				// rejected as a duplicate key (mapped to port.ErrConflict) instead of
				// producing a second document.
				{
					Keys: bson.D{
						{Key: "metadata.instanceUid", Value: 1},
					},
					Options: options.Index().SetUnique(true),
				},
				// Hashed shard-key index (see shardedCollections). Kept alongside the
				// unique index above so shardCollection has a matching index on a
				// possibly-non-empty collection; unused on non-sharded deployments.
				{
					Keys: bson.D{
						{Key: "metadata.instanceUid", Value: "hashed"},
					},
					Options: nil,
				},
				{
					Keys: bson.D{
						{Key: "metadata.namespace", Value: 1},
					},
					Options: nil,
				},
				{
					Keys: bson.D{
						{Key: "metadata.description.identifyingAttributes.key", Value: 1},
						{Key: "metadata.description.identifyingAttributes.value", Value: 1},
					},
					Options: nil,
				},
				{
					Keys: bson.D{
						{Key: "metadata.description.nonIdentifyingAttributes.key", Value: 1},
						{Key: "metadata.description.nonIdentifyingAttributes.value", Value: 1},
					},
					Options: nil,
				},
				// Index for status.connected field - used in AgentGroup statistics aggregation
				{
					Keys: bson.D{
						{Key: "status.connected", Value: 1},
					},
					Options: nil,
				},
				// Index for status.componentHealth.healthy field - used in AgentGroup statistics aggregation
				{
					Keys: bson.D{
						{Key: "status.componentHealth.healthy", Value: 1},
					},
					Options: nil,
				},
			},
		},
		{
			collectionName: agentGroupCollectionName,
			indexes: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "metadata.namespace", Value: 1},
						{Key: "metadata.name", Value: 1},
					},
					Options: nil,
				},
				{
					Keys: bson.D{
						{Key: "metadata.namespace", Value: 1},
					},
					Options: nil,
				},
				nameSearchIndex(),
			},
		},
		{
			collectionName: certificateCollectionName,
			indexes: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "metadata.namespace", Value: 1},
						{Key: "metadata.name", Value: 1},
					},
					Options: nil,
				},
				{
					Keys: bson.D{
						{Key: "metadata.namespace", Value: 1},
					},
					Options: nil,
				},
				nameSearchIndex(),
			},
		},
		{
			collectionName: agentPackageCollectionName,
			indexes: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "metadata.namespace", Value: 1},
						{Key: "metadata.name", Value: 1},
					},
					Options: nil,
				},
				{
					Keys: bson.D{
						{Key: "metadata.namespace", Value: 1},
					},
					Options: nil,
				},
				nameSearchIndex(),
				{
					Keys:    bson.D{{Key: "spec.packageType", Value: 1}},
					Options: nil,
				},
				{
					Keys:    bson.D{{Key: "spec.version", Value: 1}},
					Options: nil,
				},
			},
		},
		{
			collectionName: agentRemoteConfigCollectionName,
			indexes: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "metadata.namespace", Value: 1},
						{Key: "metadata.name", Value: 1},
					},
					Options: nil,
				},
				{
					Keys: bson.D{
						{Key: "metadata.namespace", Value: 1},
					},
					Options: nil,
				},
				nameSearchIndex(),
			},
		},
		{
			collectionName: namespaceCollectionName,
			indexes: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "metadata.name", Value: 1},
					},
					Options: nil,
				},
			},
		},
		{
			collectionName: remoteConfigSchemaCollectionName,
			indexes: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "metadata.namespace", Value: 1},
						{Key: "metadata.name", Value: 1},
					},
					Options: nil,
				},
				nameSearchIndex(),
				{
					Keys:    bson.D{{Key: "spec.binary", Value: 1}},
					Options: nil,
				},
				{
					Keys:    bson.D{{Key: "spec.version", Value: 1}},
					Options: nil,
				},
			},
		},
		{
			collectionName: endpointCollectionName,
			indexes: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "metadata.namespace", Value: 1},
						{Key: "metadata.name", Value: 1},
					},
					Options: nil,
				},
				nameSearchIndex(),
				{
					Keys:    bson.D{{Key: "spec.protocol", Value: 1}},
					Options: nil,
				},
			},
		},
		{
			collectionName: hostCollectionName,
			indexes:        platformScopedIndexes(),
		},
		{
			collectionName: containerCollectionName,
			indexes:        platformScopedIndexes(),
		},
		{
			collectionName: serverCollectionName,
			indexes: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "serverId", Value: 1},
					},
					Options: nil,
				},
			},
		},
		{
			collectionName: serverConnectionCollectionName,
			indexes: []mongo.IndexModel{
				// Backs the incremental upsert (ReplaceOne by uid) and the delete-by-uid path
				// in SyncServerConnections. Not unique: connection UIDs are already unique by
				// construction (one owning server each), and a unique index would abort
				// EnsureSchema at startup if pre-upgrade data happened to contain a duplicate.
				{
					Keys:    bson.D{{Key: "uid", Value: 1}},
					Options: nil,
				},
				// Hashed shard-key index (see shardedCollections); must exist before
				// shardCollection runs. Unused on non-sharded deployments.
				{
					Keys:    bson.D{{Key: "uid", Value: "hashed"}},
					Options: nil,
				},
				// Backs RemoveServer's per-server delete and the cluster-list query's
				// namespace + owning-server filter.
				{
					Keys: bson.D{
						{Key: "namespace", Value: 1},
						{Key: "serverId", Value: 1},
					},
					Options: nil,
				},
			},
		},
		{
			collectionName: serverHeartbeatCollectionName,
			indexes: []mongo.IndexModel{
				// One heartbeat per server; backs the upsert and the liveness lookup.
				{
					Keys:    bson.D{{Key: "serverId", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
				// TTL index: a crashed server's heartbeat (and thus its cluster-view
				// visibility) expires automatically once it stops refreshing.
				{
					Keys:    bson.D{{Key: "lastSeenAt", Value: 1}},
					Options: options.Index().SetExpireAfterSeconds(int32(serverHeartbeatTTL.Seconds())),
				},
			},
		},
	}
}

// identityIndexes is the index plan for the identity and RBAC collections: users,
// roles and role bindings. It is split out of managedIndexes so neither function
// grows past the point where the whole schema stops being readable at a glance.
//
// Like managedIndexes, it is built fresh on each call: the driver mutates the
// option builders it is handed, so a shared plan would be a data race between
// concurrent EnsureSchema calls.
func identityIndexes() []collectionAndIndexes {
	return []collectionAndIndexes{
		{
			collectionName: userCollectionName,
			indexes: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "spec.email", Value: 1},
					},
					// Case-insensitive collation so this index backs the collated GetUserByEmail
					// lookups (same emailCollation), which run on every login and every
					// authenticated request's user resolution — otherwise they full-scan users.
					Options: options.Index().SetCollation(emailCollation),
				},
				{
					// Backs GetUserByUsername (exact, case-sensitive), used by basic-auth login.
					Keys:    bson.D{{Key: "spec.username", Value: 1}},
					Options: nil,
				},
			},
		},
		{
			collectionName: roleCollectionName,
			indexes: []mongo.IndexModel{
				{
					// Backs the key lookup every GetRole issues.
					Keys:    bson.D{{Key: "metadata.uid", Value: 1}},
					Options: nil,
				},
				{
					// Backs GetRoleByName and the "?name=" prefix range scan: a role's
					// name is the display name it is referenced by.
					Keys:    bson.D{{Key: "spec.displayName", Value: 1}},
					Options: nil,
				},
				{
					Keys:    bson.D{{Key: "spec.isBuiltIn", Value: 1}},
					Options: nil,
				},
			},
		},
		{
			collectionName: roleBindingCollectionName,
			indexes: []mongo.IndexModel{
				{
					// Backs the compound (namespace, name) filter every get, put and
					// delete uses, and the namespace scoping of the listing.
					Keys: bson.D{
						{Key: "metadata.namespace", Value: 1},
						{Key: "metadata.name", Value: 1},
					},
					Options: nil,
				},
				nameSearchIndex(),
				{
					// Backs "which bindings grant this role", the one field a binding
					// advertises as selectable.
					Keys:    bson.D{{Key: "spec.roleRef.name", Value: 1}},
					Options: nil,
				},
			},
		},
	}
}

// nameSearchIndex backs the "?name=" prefix range scan, which every selector-aware
// listing of a named resource issues. It is a plain ascending index on the name:
// a prefix search is a [prefix, prefixUpperBound) range, so an index on the field
// alone serves it without the namespace having to be pinned.
//
// Label selectors have no matching entry here on purpose. A label key is chosen by
// the client, so only a wildcard index could serve arbitrary keys, and the
// collections whose labels live in a map are all config-scale — an operator has
// tens of endpoints or packages, not millions. The one fleet-scale label-filtered
// collection is agents, whose identifying attributes are already covered by the
// compound {key, value} index above.
//
// It is a function, not a variable, for the same reason managedIndexes is: the
// driver mutates the models it is handed, so a shared instance would be a data
// race between two concurrent EnsureSchema calls.
func nameSearchIndex() mongo.IndexModel {
	return mongo.IndexModel{
		Keys:    bson.D{{Key: "metadata.name", Value: 1}},
		Options: nil,
	}
}

// platformScopedIndexes is the index plan shared by hosts and containers: both are
// keyed by metadata.id, searched by metadata.name, and filtered on spec.platform.
func platformScopedIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "metadata.id", Value: 1}},
			Options: nil,
		},
		nameSearchIndex(),
		{
			Keys:    bson.D{{Key: "spec.platform", Value: 1}},
			Options: nil,
		},
	}
}

// EnsureSchema ensures that the necessary collections and indexes exist in the MongoDB database.
// This function should be called during application startup.
//
// When sharding is true the database is additionally prepared for a sharded cluster:
// sharding is enabled on the database and the collections in the built-in shard-key
// plan (shardedCollections) are sharded. The operation is idempotent — re-running it
// against an already-sharded cluster is a no-op.
func EnsureSchema(
	ctx context.Context,
	database *mongo.Database,
	sharding bool,
) error {
	err := createNonExistingCollections(ctx, database, collections)
	if err != nil {
		return fmt.Errorf("failed to create non-existing collections: %w", err)
	}

	err = createIndexes(ctx, database, managedIndexes())
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	if sharding {
		err = ensureSharding(ctx, database, shardedCollections)
		if err != nil {
			return fmt.Errorf("failed to ensure sharding: %w", err)
		}
	}

	return nil
}

func createNonExistingCollections(
	ctx context.Context,
	database *mongo.Database,
	collections []string,
) error {
	existingCollections, err := database.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("failed to list existing collections: %w", err)
	}

	notExistingCollections := lo.Filter(collections, func(c string, _ int) bool {
		return !lo.Contains(existingCollections, c)
	})

	for _, collectionName := range notExistingCollections {
		err := database.CreateCollection(ctx, collectionName)
		if err != nil {
			// Ignore NamespaceExists error (code 48) which can occur in distributed setups
			// when multiple servers try to create the same collection concurrently
			var cmdErr mongo.CommandError
			if errors.As(err, &cmdErr) && cmdErr.Code == 48 { // NamespaceExists
				continue
			}

			return fmt.Errorf("failed to create collection %s: %w", collectionName, err)
		}
	}

	return nil
}

type collectionAndIndexes struct {
	collectionName string
	indexes        []mongo.IndexModel
}

func createIndexes(
	ctx context.Context,
	database *mongo.Database,
	indexes []collectionAndIndexes,
) error {
	for _, ci := range indexes {
		collection := database.Collection(ci.collectionName)

		_, err := collection.Indexes().CreateMany(ctx, ci.indexes)
		if err != nil {
			return fmt.Errorf("failed to create indexes for collection %s: %w", ci.collectionName, err)
		}
	}

	return nil
}

// shardCollectionSpec describes how a single collection is sharded.
type shardCollectionSpec struct {
	collectionName string
	// shardKey is the shardCollection key document, e.g. {"uid": "hashed"}.
	shardKey bson.D
}

// ensureSharding enables sharding on the database and shards the planned collections.
// It is idempotent: enabling sharding on an already-enabled database and re-sharding
// an already-sharded collection with the same key are both treated as success.
func ensureSharding(
	ctx context.Context,
	database *mongo.Database,
	specs []shardCollectionSpec,
) error {
	admin := database.Client().Database("admin")

	err := runAdminCommand(ctx, admin, bson.D{{Key: "enableSharding", Value: database.Name()}})
	if err != nil {
		return fmt.Errorf("failed to enable sharding on database %s: %w", database.Name(), err)
	}

	for _, spec := range specs {
		namespace := database.Name() + "." + spec.collectionName

		err := runAdminCommand(ctx, admin, bson.D{
			{Key: "shardCollection", Value: namespace},
			{Key: "key", Value: spec.shardKey},
		})
		if err != nil {
			return fmt.Errorf("failed to shard collection %s: %w", namespace, err)
		}
	}

	return nil
}

// runAdminCommand runs a sharding admin command, tolerating the errors that make
// the operation idempotent (already-enabled / already-sharded) the same way
// createNonExistingCollections tolerates NamespaceExists.
func runAdminCommand(ctx context.Context, admin *mongo.Database, command bson.D) error {
	err := admin.RunCommand(ctx, command).Err()
	if err != nil && !isAlreadyShardedError(err) {
		return fmt.Errorf("admin command failed: %w", err)
	}

	return nil
}

// isAlreadyShardedError reports whether err indicates the database/collection is
// already in the desired sharded state, which is safe to ignore for idempotency.
func isAlreadyShardedError(err error) bool {
	var cmdErr mongo.CommandError
	if !errors.As(err, &cmdErr) {
		return false
	}

	// AlreadyInitialized (23) is returned when re-sharding a collection with the
	// same key; the message check covers server versions that report it as a plain
	// command error instead.
	const alreadyInitialized = 23
	if cmdErr.Code == alreadyInitialized {
		return true
	}

	return strings.Contains(cmdErr.Message, "already sharded")
}
