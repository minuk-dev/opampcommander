package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"slices"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var (
	// ErrIDFieldNotExist is returned when the ID field does not exist in the entity.
	ErrIDFieldNotExist = errors.New("_id field does not exist in the entity")
)

// KeyFunc is a function that generates a unique key for a given domain model.
type KeyFunc[Entity any, KeyType any] func(domain *Entity) KeyType

type commonEntityAdapter[Entity any, KeyType any] struct {
	logger             *slog.Logger
	collection         *mongo.Collection
	KeyFunc            KeyFunc[Entity, KeyType]
	KeyQueryFunc       func(key KeyType) any
	keyFieldName       string
	deletedAtFieldName string
	selectors          selectorSchema
}

// newCommonAdapter wires the shared get/list/put behaviour for one collection.
// selectors is that collection's server-side filtering schema; pass
// noSelectorSchema for a collection that supports none, which makes a request
// carrying a selector fail rather than come back unfiltered.
func newCommonAdapter[Entity any, KeyType any](
	logger *slog.Logger,
	collection *mongo.Collection,
	keyFieldName string,
	keyFunc KeyFunc[Entity, KeyType],
	keyQueryFunc func(key KeyType) any,
	selectors selectorSchema,
) commonEntityAdapter[Entity, KeyType] {
	return commonEntityAdapter[Entity, KeyType]{
		logger:             logger,
		collection:         collection,
		keyFieldName:       keyFieldName,
		KeyFunc:            keyFunc,
		KeyQueryFunc:       keyQueryFunc,
		deletedAtFieldName: "metadata.deletedAt",
		selectors:          selectors,
	}
}

func (a *commonEntityAdapter[Entity, KeyType]) get(
	ctx context.Context,
	key KeyType,
	options *model.GetOptions,
) (*Entity, error) {
	var filter bson.M
	if options != nil && options.IncludeDeleted {
		filter = a.filterByKey(key)
	} else {
		filter = a.filterByKeyExcludingDeleted(key)
	}

	result := a.collection.FindOne(ctx, filter)

	err := result.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, model.ErrResourceNotExist
		}

		return nil, fmt.Errorf("failed to get resource from mongodb: %w", err)
	}

	var entity Entity

	err = result.Decode(&entity)
	if err != nil {
		return nil, fmt.Errorf("failed to decode resource from mongodb: %w", err)
	}

	return &entity, nil
}

func (a *commonEntityAdapter[Entity, KeyType]) list(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*Entity], error) {
	return a.listWithConditions(ctx, options)
}

// listWithConditions lists the entities matching the soft-delete filter, the
// collection's selector schema applied to options, and the caller's own extra
// conditions — all AND-ed together.
//
// The conditions are AND-ed as a list rather than merged into one map because a
// selector can produce several conditions on the same field ($elemMatch, $nor),
// which a map merge would silently collapse to the last one.
//
// Every filter goes into the query the page is cut from, so RemainingItemCount
// counts the same set the page was drawn from.
func (a *commonEntityAdapter[Entity, KeyType]) listWithConditions(
	ctx context.Context,
	options *model.ListOptions,
	extraConditions ...bson.M,
) (*model.ListResponse[*Entity], error) {
	if options == nil {
		//exhaustruct:ignore
		options = &model.ListOptions{}
	}

	continueTokenObjectID, err := bson.ObjectIDFromHex(options.Continue)
	if err != nil && options.Continue != "" {
		return nil, fmt.Errorf("invalid continue token: %w", err)
	}

	conditions := slices.Clone(extraConditions)

	if !options.IncludeDeleted {
		if excludeDeleted := a.excludeDeletedFilter(); excludeDeleted != nil {
			conditions = append(conditions, excludeDeleted)
		}
	}

	selectorConditions, err := a.selectors.conditions(options)
	if err != nil {
		return nil, err
	}

	conditions = append(conditions, selectorConditions...)

	prefix := mongo.Pipeline{bson.D{{Key: "$match", Value: buildFilter(conditions)}}}

	entities, continueToken, remaining, err := aggregateListPage[Entity](
		ctx, a.logger, a.collection, prefix, continueTokenObjectID, options.Limit,
	)
	if err != nil {
		return nil, err
	}

	return &model.ListResponse[*Entity]{
		Items:              entities,
		Continue:           continueToken,
		RemainingItemCount: remaining,
	}, nil
}

// facetPage is the single document a $facet emits: the page plus a one-element count array
// ($count emits nothing when zero documents match).
type facetPage[Entity any] struct {
	Items []*Entity `bson:"items"`
	Count []struct {
		Count int64 `bson:"count"`
	} `bson:"count"`
}

// aggregateListPage runs a single-snapshot paginated list. prefix selects the candidate
// documents (a $match, plus e.g. a $lookup); a $facet then returns the page and the total count
// in one snapshot, so RemainingItemCount cannot skew against the page the way a separate find +
// CountDocuments can. The single round-trip is also session-safe inside a transaction.
//
// The _id-ascending sort makes the {_id: {$gt: token}} continue-token scheme stable and lets a
// sharded mongos merge-sort shard results; it is required for correctness, not just determinism.
// A non-positive limit means "no limit".
func aggregateListPage[Entity any](
	ctx context.Context,
	logger *slog.Logger,
	collection *mongo.Collection,
	prefix mongo.Pipeline,
	continueToken bson.ObjectID,
	limit int64,
) ([]*Entity, string, int64, error) {
	pipeline := slices.Clone(prefix)

	if continueTokenFilter := withContinueToken(continueToken); continueTokenFilter != nil {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: continueTokenFilter}})
	}

	itemStages := bson.A{bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}}}
	if limit > 0 {
		itemStages = append(itemStages, bson.D{{Key: "$limit", Value: limit}})
	}

	pipeline = append(pipeline, bson.D{{Key: "$facet", Value: bson.D{
		{Key: "items", Value: itemStages},
		{Key: "count", Value: bson.A{bson.D{{Key: "$count", Value: "count"}}}},
	}}})

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to list resources from mongodb: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var results []facetPage[Entity]

	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to decode resources from mongodb: %w", err)
	}

	// $facet emits exactly one document; a missing one means an empty page.
	if len(results) == 0 {
		return nil, "", 0, nil
	}

	page := results[0]

	continueTokenStr, err := getContinueTokenFromEntities(page.Items)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to get continue token from entities: %w", err)
	}

	var count int64
	if len(page.Count) > 0 {
		count = page.Count[0].Count
	}

	return page.Items, continueTokenStr, count - int64(len(page.Items)), nil
}

func (a *commonEntityAdapter[Entity, KeyType]) put(ctx context.Context, entity *Entity) error {
	_, err := a.collection.ReplaceOne(ctx,
		a.filterByKey(a.KeyFunc(entity)),
		entity,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("failed to put resource to mongodb: %w", err)
	}

	return nil
}

// casReplace performs an optimistic-concurrency upsert of doc, identified by
// keyFilter, whose resourceVersion field has already been set to the next version.
//
// expected is the version the caller loaded doc at. When it is greater than 0 the
// write matches only a stored document still at that version, so a concurrent
// writer that advanced it makes casReplace return [model.ErrConflict] instead of
// silently clobbering the change. expected 0 is a create (or the first write of a
// pre-optimistic-concurrency document that has no resourceVersion field yet): it
// upserts by key alone, so a legacy document is migrated rather than duplicated.
//
// If the collection carries a unique index on the logical key, a racing create is
// rejected as a duplicate key, which casReplace also surfaces as [model.ErrConflict].
func casReplace(
	ctx context.Context,
	collection *mongo.Collection,
	keyFilter bson.M,
	doc any,
	expected int64,
) error {
	filter := maps.Clone(keyFilter)
	if expected != 0 {
		filter[resourceVersionFieldName] = expected
	}

	result, err := collection.ReplaceOne(ctx, filter, doc, options.Replace().SetUpsert(expected == 0))
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%w: resource was created concurrently", model.ErrConflict)
		}

		return fmt.Errorf("failed to put resource to mongodb: %w", err)
	}

	// No matched (and not freshly upserted) document means the version filter did not
	// find the expected version — another writer advanced it (or deleted the resource).
	if result.MatchedCount == 0 && result.UpsertedCount == 0 {
		return fmt.Errorf("%w: resource was modified concurrently", model.ErrConflict)
	}

	return nil
}

// deleteOne permanently removes the document identified by key. It returns
// model.ErrResourceNotExist when no matching document is found.
//
// WARNING: this is a HARD delete — it issues a real DeleteOne and does NOT apply
// the soft-delete filter (excludeDeletedFilter) that get/list use. Most resources
// in this codebase soft-delete by stamping metadata.deletedAt and rely on the
// tombstone surviving (audit, IncludeDeleted reads, login flow). Only use this for
// resource types that have no deletedAt field / no soft-delete semantics (today:
// agents). Wiring it for a soft-deleted resource will silently purge tombstones.
func (a *commonEntityAdapter[Entity, KeyType]) deleteOne(ctx context.Context, key KeyType) error {
	result, err := a.collection.DeleteOne(ctx, a.filterByKey(key))
	if err != nil {
		return fmt.Errorf("failed to delete resource from mongodb: %w", err)
	}

	if result.DeletedCount == 0 {
		return model.ErrResourceNotExist
	}

	return nil
}

func getContinueTokenFromEntities[Entity any](entities []*Entity) (string, error) {
	if len(entities) == 0 {
		return "", nil
	}

	lastEntity := entities[len(entities)-1]
	idField := reflect.ValueOf(lastEntity).Elem().FieldByName("ID")

	idFieldValue, ok := idField.Interface().(*bson.ObjectID)
	if !ok {
		return "", ErrIDFieldNotExist
	}

	return idFieldValue.Hex(), nil
}

func (a *commonEntityAdapter[Domain, KeyType]) filterByKey(key KeyType) bson.M {
	return bson.M{a.keyFieldName: a.KeyQueryFunc(key)}
}

func (a *commonEntityAdapter[Domain, KeyType]) filterByKeyExcludingDeleted(key KeyType) bson.M {
	return combineFilters(a.filterByKey(key), a.excludeDeletedFilter())
}

func (a *commonEntityAdapter[Domain, KeyType]) excludeDeletedFilter() bson.M {
	if a.deletedAtFieldName == "" {
		return nil
	}

	return bson.M{a.deletedAtFieldName: nil}
}

func combineFilters(filters ...bson.M) bson.M {
	result := bson.M{}

	for _, filter := range filters {
		maps.Copy(result, filter)
	}

	return result
}

func withContinueToken(continueToken bson.ObjectID) bson.M {
	if continueToken == bson.NilObjectID {
		return nil
	}

	return bson.M{"_id": bson.M{"$gt": continueToken}}
}
