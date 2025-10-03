package mongodb

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

// KeyFunc is a function that generates a unique key for a given domain model.
type KeyFunc[Entity any, KeyType any] func(domain *Entity) KeyType

type commonEntityAdapter[Entity any, KeyType any] struct {
	collection   *mongo.Collection
	KeyFunc      KeyFunc[Entity, KeyType]
	keyFieldName string
}

func newCommonAdapter[Entity any, KeyType any](
	collection *mongo.Collection,
	keyFieldName string,
	keyFunc KeyFunc[Entity, KeyType],
) commonEntityAdapter[Entity, KeyType] {
	return commonEntityAdapter[Entity, KeyType]{
		collection:   collection,
		keyFieldName: keyFieldName,
		KeyFunc:      keyFunc,
	}
}

func (a *commonEntityAdapter[Entity, KeyType]) get(ctx context.Context, key KeyType) (*Entity, error) {
	result := a.collection.FindOne(ctx, a.filterByKey(key))
	err := result.Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
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

func (a *commonEntityAdapter[Entity, KeyType]) list(ctx context.Context, options *domainmodel.ListOptions) (*domainmodel.ListResponse[*Entity], error) {
	var count int64
	var continueToken string
	var entities []*Entity
	var wg sync.WaitGroup
	if options == nil {
		options = &domainmodel.ListOptions{}
	}

	var fErr error
	var lErr error
	wg.Go(func() {
		cursor, err := a.collection.Find(ctx,
			withContinueToken(options.Continue),
			withLimit(options.Limit),
		)
		if err != nil {
			fErr = fmt.Errorf("failed to list resources from mongodb: %w", err)
			return
		}

		defer cursor.Close(ctx)

		err = cursor.All(ctx, &entities)
		if err != nil {
			fErr = fmt.Errorf("failed to decode resources from mongodb: %w", err)
			return
		}
		if len(entities) == 0 {
			return
		}

		lastEntity := entities[len(entities)-1]
		idField := reflect.ValueOf(lastEntity).Elem().FieldByName("ID")
		idFieldValue := idField.Interface().(*primitive.ObjectID)
		continueToken = idFieldValue.String()
	})
	wg.Go(func() {
		cnt, err := a.collection.CountDocuments(ctx, withContinueToken(options.Continue))
		if err != nil {
			lErr = fmt.Errorf("failed to count resources in mongodb: %w", err)
			return
		}
		count = cnt
	})
	wg.Wait()

	if fErr != nil || lErr != nil {
		return nil, fmt.Errorf("list operation failed: %v %v", fErr, lErr)
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

func (a *commonEntityAdapter[Domain, KeyType]) filterByKey(key KeyType) bson.M {
	return filterByField(a.keyFieldName, key)
}

func filterByField(field string, value any) bson.M {
	return bson.M{field: value}
}

func withContinueToken(continueToken string) bson.M {
	if continueToken == "" {
		return bson.M{}
	}

	objectID, err := primitive.ObjectIDFromHex(continueToken)
	if err != nil {
		panic(fmt.Sprintf("invalid continue token: %v", err))
	}

	return bson.M{"_id": bson.M{"$gt": objectID}}
}

func withLimit(limit int64) *options.FindOptions {
	if limit <= 0 {
		return nil
	}

	return options.Find().SetLimit(limit)
}
