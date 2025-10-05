package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var (
	// ErrIDFieldNotExist is returned when the ID field does not exist in the entity.
	ErrIDFieldNotExist = errors.New("_id field does not exist in the entity")
)

// KeyFunc is a function that generates a unique key for a given domain model.
type KeyFunc[Entity any, KeyType any] func(domain *Entity) KeyType

type commonEntityAdapter[Entity any, KeyType any] struct {
	logger       *slog.Logger
	collection   *mongo.Collection
	KeyFunc      KeyFunc[Entity, KeyType]
	keyFieldName string
}

func newCommonAdapter[Entity any, KeyType any](
	logger *slog.Logger,
	collection *mongo.Collection,
	keyFieldName string,
	keyFunc KeyFunc[Entity, KeyType],
) commonEntityAdapter[Entity, KeyType] {
	return commonEntityAdapter[Entity, KeyType]{
		logger:       logger,
		collection:   collection,
		keyFieldName: keyFieldName,
		KeyFunc:      keyFunc,
	}
}

func (a *commonEntityAdapter[Entity, KeyType]) get(ctx context.Context, key KeyType) (*Entity, error) {
	result := a.collection.FindOne(ctx, a.filterByKey(key))

	err := result.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domainport.ErrResourceNotExist
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
	options *domainmodel.ListOptions,
) (*domainmodel.ListResponse[*Entity], error) {
	var (
		count         int64
		continueToken string
		entities      []*Entity
		queryWg       sync.WaitGroup
	)

	if options == nil {
		//exhaustruct:ignore
		options = &domainmodel.ListOptions{}
	}

	var (
		fErr error
		lErr error
	)

	continueTokenObjectID, err := primitive.ObjectIDFromHex(options.Continue)
	if err != nil && options.Continue != "" {
		return nil, fmt.Errorf("invalid continue token: %w", err)
	}

	queryWg.Go(func() {
		entities, err := listWithContinueTokenAndLimit[Entity](ctx, a.logger, a.collection, options.Continue, options.Limit)
		if err != nil {
			fErr = fmt.Errorf("failed to list resources from mongodb: %w", err)

			return
		}

		continueToken, err = getContinueTokenFromEntities(entities)
		if err != nil {
			fErr = fmt.Errorf("failed to get continue token from entities: %w", err)

			return
		}
	})
	queryWg.Go(func() {
		cnt, err := a.collection.CountDocuments(ctx, withContinueToken(continueTokenObjectID))
		if err != nil {
			lErr = fmt.Errorf("failed to count resources in mongodb: %w", err)

			return
		}

		count = cnt
	})
	queryWg.Wait()

	if fErr != nil || lErr != nil {
		return nil, fmt.Errorf("list operation failed: %w %w", fErr, lErr)
	}

	return &domainmodel.ListResponse[*Entity]{
		Items:              entities,
		Continue:           continueToken,
		RemainingItemCount: count - int64(len(entities)),
	}, nil
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

func listWithContinueTokenAndLimit[Entity any](
	ctx context.Context,
	logger *slog.Logger,
	collection *mongo.Collection,
	continueToken string,
	limit int64,
) ([]*Entity, error) {
	continueTokenObjectID, err := primitive.ObjectIDFromHex(continueToken)
	if err != nil && continueToken != "" {
		return nil, fmt.Errorf("invalid continue token: %w", err)
	}

	cursor, err := collection.Find(ctx,
		withContinueToken(continueTokenObjectID),
		withLimit(limit),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources from mongodb: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
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
	idField := reflect.ValueOf(lastEntity).Elem().FieldByName("_id")

	idFieldValue, ok := idField.Interface().(*primitive.ObjectID)
	if !ok {
		return "", ErrIDFieldNotExist
	}

	return idFieldValue.Hex(), nil
}

func (a *commonEntityAdapter[Domain, KeyType]) filterByKey(key KeyType) bson.M {
	return filterByField(a.keyFieldName, key)
}

func filterByField(field string, value any) bson.M {
	return bson.M{field: value}
}

func withContinueToken(continueToken primitive.ObjectID) bson.M {
	if continueToken == primitive.NilObjectID {
		return nil
	}

	return bson.M{"_id": bson.M{"$gt": continueToken}}
}

func withLimit(limit int64) *options.FindOptions {
	if limit <= 0 {
		return nil
	}

	return options.Find().SetLimit(limit)
}
