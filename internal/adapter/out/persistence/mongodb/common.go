package mongodb

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

// ToEntityFunc is a function that converts a domain model to its corresponding entity representation.
type ToEntityFunc[Domain any] func(domain *Domain) (Entity[Domain], error)

// KeyFunc is a function that generates a unique key for a given domain model.
type KeyFunc[Domain any] func(domain *Domain) string

// Entity is a generic interface that defines a method to convert an entity to its corresponding domain model.
type Entity[Domain any] interface {
	ToDomain() *Domain
}

type commonAdapter[Domain any] struct {
	collection            *mongo.Collection
	CreateEmptyEntityFunc func() Entity[Domain]
	ToEntityFunc          ToEntityFunc[Domain]
	KeyFunc               KeyFunc[Domain]
}

func newCommonAdapter[Domain any](
	collection *mongo.Collection,
	toEntityFunc ToEntityFunc[Domain],
	newEmptyEntityFunc func() Entity[Domain],
	keyFunc KeyFunc[Domain],
) commonAdapter[Domain] {
	return commonAdapter[Domain]{
		collection:            collection,
		CreateEmptyEntityFunc: newEmptyEntityFunc,
		ToEntityFunc:          toEntityFunc,
		KeyFunc:               keyFunc,
	}
}

func (a *commonAdapter[Domain]) get(ctx context.Context, resourceId string) (*Domain, error) {
	result := a.collection.FindOne(ctx, filterByResourceId(resourceId))
	err := result.Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domainport.ErrResourceNotExist
		}
		return nil, fmt.Errorf("failed to get resource from mongodb: %w", err)
	}

	var entity = a.CreateEmptyEntityFunc()
	err = result.Decode(&entity)
	if err != nil {
		return nil, fmt.Errorf("failed to decode resource from mongodb: %w", err)
	}

	domain := entity.ToDomain()
	return domain, nil
}

func (a *commonAdapter[Domain]) list(ctx context.Context, options *domainmodel.ListOptions) (*domainmodel.ListResponse[*Domain], error) {
	var count int64
	var continueToken string
	var domains []*Domain
	var wg sync.WaitGroup

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

		for cursor.Next(ctx) {
			var entity = a.CreateEmptyEntityFunc()
			err := cursor.Decode(&entity)
			if err != nil {
				fErr = fmt.Errorf("failed to decode resource from mongodb: %w", err)
				return
			}

			domain := entity.ToDomain()
			domains = append(domains, domain)
		}

		if len(domains) > 0 {
			lastDomain := domains[len(domains)-1]
			continueToken = a.KeyFunc(lastDomain)
		}
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

	return &domainmodel.ListResponse[*Domain]{
		Items:              domains,
		Continue:           continueToken,
		RemainingItemCount: count - int64(len(domains)),
	}, nil
}

func (a *commonAdapter[Domain]) put(ctx context.Context, domain *Domain) error {
	entity, err := a.ToEntityFunc(domain)
	if err != nil {
		return fmt.Errorf("failed to convert domain model to entity: %w", err)
	}

	_, err = a.collection.ReplaceOne(ctx,
		filterByResourceId(a.KeyFunc(domain)),
		entity,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("failed to put resource to mongodb: %w", err)
	}

	return nil
}

func filterById(id primitive.ObjectID) bson.M {
	return filterByField("_id", id)
}

func filterByResourceId(resourceId string) bson.M {
	return filterByField("resourceId", resourceId)
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
