package mongodb

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestEnsureShardingRejectsDifferentKey(t *testing.T) {
	t.Parallel()

	const message = "Collection 'commander.agent_enrollments' already exists with a different shard key."

	err := ensureSharding("commander", shardedCollections, func(command bson.D) error {
		if command[0] == (bson.E{Key: "shardCollection", Value: "commander.agent_enrollments"}) {
			return mongo.CommandError{Code: 23, Message: message}
		}

		return nil
	})
	require.ErrorContains(t, err, message)
}

// TestShardedCollectionsAreManaged verifies every collection in the shard-key
// plan is also in the managed collection set, so it is created explicitly before
// shardCollection runs against it.
func TestShardedCollectionsAreManaged(t *testing.T) {
	t.Parallel()

	for _, spec := range shardedCollections {
		assert.Contains(t, collections, spec.collectionName,
			"sharded collection %q must be in the managed collections list", spec.collectionName)
	}
}

// TestShardKeyHasMatchingHashedIndex verifies each hashed shard key has a matching
// hashed index declared in the managed indexes. shardCollection requires an index
// matching the shard key to already exist on a non-empty collection.
func TestShardKeyHasMatchingHashedIndex(t *testing.T) {
	t.Parallel()

	for _, spec := range shardedCollections {
		field, isHashed := hashedShardKeyField(spec.shardKey)
		if !isHashed {
			continue
		}

		assert.Truef(t, hasIndex(spec.collectionName, bson.D{{Key: field, Value: "hashed"}}),
			"sharded collection %q needs a hashed index on %q", spec.collectionName, field)
	}
}

// TestUniqueIndexPrefixedByShardKey enforces MongoDB's rule that a unique index on
// a sharded collection must be prefixed by the shard key. For our hashed single-field
// shard keys this means every unique index's first field equals the shard-key field.
func TestUniqueIndexPrefixedByShardKey(t *testing.T) {
	t.Parallel()

	for _, spec := range shardedCollections {
		field, isHashed := hashedShardKeyField(spec.shardKey)
		if !isHashed {
			continue
		}

		for _, ci := range managedIndexes() {
			if ci.collectionName != spec.collectionName {
				continue
			}

			for _, idx := range ci.indexes {
				if !isUniqueIndex(t, idx) {
					continue
				}

				keys, ok := idx.Keys.(bson.D)
				if assert.Truef(t, ok && len(keys) > 0, "unique index on %q has no keys", spec.collectionName) {
					assert.Equalf(t, field, keys[0].Key,
						"unique index on sharded collection %q must be prefixed by shard key %q",
						spec.collectionName, field)
				}
			}
		}
	}
}

// isUniqueIndex reports whether the index model sets the unique option, by
// applying the options builder (the v2 driver exposes options only as setters).
func isUniqueIndex(t *testing.T, idx mongo.IndexModel) bool {
	t.Helper()

	if idx.Options == nil {
		return false
	}

	var opts options.IndexOptions
	for _, set := range idx.Options.List() {
		require.NoError(t, set(&opts))
	}

	return opts.Unique != nil && *opts.Unique
}

// hashedShardKeyField returns the single field of a hashed shard key.
func hashedShardKeyField(key bson.D) (string, bool) {
	if len(key) != 1 {
		return "", false
	}

	if v, ok := key[0].Value.(string); ok && v == "hashed" {
		return key[0].Key, true
	}

	return "", false
}

func hasIndex(collectionName string, keys bson.D) bool {
	for _, ci := range managedIndexes() {
		if ci.collectionName != collectionName {
			continue
		}

		if lo.ContainsBy(ci.indexes, func(idx mongo.IndexModel) bool {
			k, ok := idx.Keys.(bson.D)

			return ok && k.String() == keys.String()
		}) {
			return true
		}
	}

	return false
}
