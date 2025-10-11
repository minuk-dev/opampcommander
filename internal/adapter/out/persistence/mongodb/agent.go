package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ domainport.AgentPersistencePort = (*AgentRepository)(nil)

const (
	agentCollectionName = "agents"
)

// AgentRepository is a struct that implements the AgentPersistencePort interface.
type AgentRepository struct {
	collection *mongo.Collection
	logger     *slog.Logger
	common     commonEntityAdapter[entity.Agent, uuid.UUID]
}

// NewAgentRepository creates a new instance of AgentRepository.
func NewAgentRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *AgentRepository {
	collection := mongoDatabase.Collection(agentCollectionName)
	keyFunc := func(domain *entity.Agent) uuid.UUID {
		return domain.Metadata.InstanceUID
	}

	return &AgentRepository{
		collection: collection,
		logger:     logger,
		common: newCommonAdapter(
			logger,
			collection,
			entity.AgentKeyFieldName,
			keyFunc,
		),
	}
}

// GetAgent implements port.AgentPersistencePort.
func (a *AgentRepository) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	entity, err := a.common.get(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent from persistence: %w", err)
	}

	return entity.ToDomain(), nil
}

// ListAgents implements port.AgentPersistencePort.
func (a *AgentRepository) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents from persistence: %w", err)
	}

	return &model.ListResponse[*model.Agent]{
		Items: lo.Map(resp.Items, func(item *entity.Agent, _ int) *model.Agent {
			return item.ToDomain()
		}),
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutAgent implements port.AgentPersistencePort.
func (a *AgentRepository) PutAgent(ctx context.Context, agent *model.Agent) error {
	entity := entity.AgentFromDomain(agent)

	err := a.common.put(ctx, entity)
	if err != nil {
		return fmt.Errorf("failed to put agent to persistence: %w", err)
	}

	return nil
}

// ListAgentsBySelector implements port.AgentPersistencePort.
//
//nolint:funlen // Reason: unavoidable.
func (a *AgentRepository) ListAgentsBySelector(
	ctx context.Context,
	selector model.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	var (
		// To prevent shadowing in goroutines, we use retval suffix.
		countRetval         int64
		continueTokenRetval string
		entitiesRetval      []*entity.Agent
	)

	continueTokenObjectID, err := bson.ObjectIDFromHex(options.Continue)
	if err != nil && options.Continue != "" {
		return nil, fmt.Errorf("invalid continue token: %w", err)
	}

	filter := bson.M{
		"$and": []bson.M{
			{
				entity.IdentifyingAttributesFieldName: bson.M{
					"$all": selector.IdentifyingAttributes,
				},
			},
			{
				entity.NonIdentifyingAttributesFieldName: bson.M{
					"$all": selector.NonIdentifyingAttributes,
				},
			},
			withContinueToken(continueTokenObjectID),
		},
	}

	var queryWg sync.WaitGroup

	var (
		fErr error
		lErr error
	)

	queryWg.Go(func() {
		cursor, err := a.collection.Find(ctx, filter, withLimit(options.Limit))
		if err != nil {
			fErr = fmt.Errorf("failed to find agents by selector from mongodb: %w", err)

			return
		}

		defer func() {
			closeErr := cursor.Close(ctx)
			if closeErr != nil {
				a.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
			}
		}()

		var entities []*entity.Agent

		err = cursor.All(ctx, &entities)
		if err != nil {
			fErr = fmt.Errorf("failed to decode agents by selector from mongodb: %w", err)

			return
		}

		continueToken, err := getContinueTokenFromEntities(entities)
		if err != nil {
			fErr = fmt.Errorf("failed to get continue token from entities: %w", err)

			return
		}

		entitiesRetval = entities
		continueTokenRetval = continueToken
	})

	queryWg.Go(func() {
		cnt, err := a.collection.CountDocuments(ctx, filter)
		if err != nil {
			lErr = fmt.Errorf("failed to count agents by selector in mongodb: %w", err)

			return
		}

		countRetval = cnt
	})

	queryWg.Wait()

	if fErr != nil || lErr != nil {
		return nil, fmt.Errorf("list by selector operation failed: %w %w", fErr, lErr)
	}

	return &model.ListResponse[*model.Agent]{
		Items: lo.Map(entitiesRetval, func(item *entity.Agent, _ int) *model.Agent {
			return item.ToDomain()
		}),
		Continue:           continueTokenRetval,
		RemainingItemCount: countRetval - int64(len(entitiesRetval)),
	}, nil
}
