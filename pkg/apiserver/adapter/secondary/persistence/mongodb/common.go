package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"sync"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
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
}

func newCommonAdapter[Entity any, KeyType any](
	logger *slog.Logger,
	collection *mongo.Collection,
	keyFieldName string,
	keyFunc KeyFunc[Entity, KeyType],
	keyQueryFunc func(key KeyType) any,
) commonEntityAdapter[Entity, KeyType] {
	return commonEntityAdapter[Entity, KeyType]{
		logger:             logger,
		collection:         collection,
		keyFieldName:       keyFieldName,
		KeyFunc:            keyFunc,
		KeyQueryFunc:       keyQueryFunc,
		deletedAtFieldName: "metadata.deletedAt",
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
			return nil, port.ErrResourceNotExist
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
	return a.listWithFilter(ctx, options, nil)
}

//nolint:funlen // Reason: unavoidable, runs find + count and assembles a list response.
func (a *commonEntityAdapter[Entity, KeyType]) listWithFilter(
	ctx context.Context,
	options *model.ListOptions,
	extraFilter bson.M,
) (*model.ListResponse[*Entity], error) {
	if options == nil {
		//exhaustruct:ignore
		options = &model.ListOptions{}
	}

	continueTokenObjectID, err := bson.ObjectIDFromHex(options.Continue)
	if err != nil && options.Continue != "" {
		return nil, fmt.Errorf("invalid continue token: %w", err)
	}

	var baseFilter bson.M
	if options.IncludeDeleted {
		baseFilter = extraFilter
	} else {
		baseFilter = combineFilters(a.excludeDeletedFilter(), extraFilter)
	}

	var (
		countRetval         int64
		continueTokenRetval string
		entitiesRetval      []*Entity
		fErr                error
		lErr                error
	)

	findTask := func() {
		entities, listErr := a.listWithContinueTokenAndLimit(
			ctx, continueTokenObjectID, options.Limit, baseFilter,
		)
		if listErr != nil {
			fErr = fmt.Errorf("failed to list resources from mongodb: %w", listErr)

			return
		}

		continueToken, tokenErr := getContinueTokenFromEntities(entities)
		if tokenErr != nil {
			fErr = fmt.Errorf("failed to get continue token from entities: %w", tokenErr)

			return
		}

		entitiesRetval = entities
		continueTokenRetval = continueToken
	}

	countTask := func() {
		filter := combineFilters(baseFilter, withContinueToken(continueTokenObjectID))

		cnt, countErr := a.collection.CountDocuments(ctx, filter)
		if countErr != nil {
			lErr = fmt.Errorf("failed to count resources in mongodb: %w", countErr)

			return
		}

		countRetval = cnt
	}

	runListQueries(ctx, findTask, countTask)

	if fErr != nil || lErr != nil {
		return nil, fmt.Errorf("list operation failed: %w %w", fErr, lErr)
	}

	return &model.ListResponse[*Entity]{
		Items:              entitiesRetval,
		Continue:           continueTokenRetval,
		RemainingItemCount: countRetval - int64(len(entitiesRetval)),
	}, nil
}

// runListQueries runs the find and count queries that back list operations.
// Outside a MongoDB session it runs them in parallel to save a round-trip.
// Inside a session (i.e. a transaction) the driver's *mongo.Session is NOT
// goroutine-safe — see https://pkg.go.dev/go.mongodb.org/mongo-driver/v2/mongo#Session
// — so we serialise to avoid session-state corruption / "transaction in progress"
// errors when a List call happens inside [port.TransactionRunner].
func runListQueries(ctx context.Context, findTask, countTask func()) {
	if mongo.SessionFromContext(ctx) != nil {
		findTask()
		countTask()

		return
	}

	var queryWg sync.WaitGroup

	queryWg.Go(findTask)
	queryWg.Go(countTask)
	queryWg.Wait()
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

// deleteOne permanently removes the document identified by key. It returns
// port.ErrResourceNotExist when no matching document is found.
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
		return port.ErrResourceNotExist
	}

	return nil
}

func (a *commonEntityAdapter[Entity, KeyType]) listWithContinueTokenAndLimit(
	ctx context.Context,
	continueTokenObjectID bson.ObjectID,
	limit int64,
	baseFilter bson.M,
) ([]*Entity, error) {
	filter := combineFilters(baseFilter, withContinueToken(continueTokenObjectID))

	cursor, err := a.collection.Find(ctx, filter, withLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to list resources from mongodb: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			a.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var entities []*Entity

	err = cursor.All(ctx, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to decode resources from mongodb: %w", err)
	}

	return entities, nil
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

func withLimit(limit int64) *options.FindOptionsBuilder {
	if limit <= 0 {
		return nil
	}

	options.Find().SetLimit(limit)

	return options.Find().SetLimit(limit)
}
