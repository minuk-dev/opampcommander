package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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
	keyFunc := func(entity *entity.Agent) uuid.UUID {
		return uuid.UUID(entity.Metadata.InstanceUID.Data)
	}
	keyQueryFunc := func(key uuid.UUID) any {
		return bson.Binary{
			Subtype: bson.TypeBinaryUUID,
			Data:    key[:],
		}
	}

	return &AgentRepository{
		collection: collection,
		logger:     logger,
		common: newCommonAdapter(
			logger,
			collection,
			entity.AgentKeyFieldName,
			keyFunc,
			keyQueryFunc,
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

	if options == nil {
		//exhaustruct:ignore
		options = &model.ListOptions{}
	}

	continueTokenObjectID, err := bson.ObjectIDFromHex(options.Continue)
	if err != nil && options.Continue != "" {
		return nil, fmt.Errorf("invalid continue token: %w", err)
	}

	allConditions := SelectorToMatchConditions(AgentSelectorToEntity(selector))

	// Add continue token condition if present
	continueTokenFilter := withContinueToken(continueTokenObjectID)
	if continueTokenFilter != nil {
		allConditions = append(allConditions, continueTokenFilter)
	}

	filter := buildFilter(allConditions)

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

// AgentSelectorToEntity converts a domain AgentSelector to a persistence entity AgentSelector.
func AgentSelectorToEntity(selector model.AgentSelector) entity.AgentSelector {
	return entity.AgentSelector{
		IdentifyingAttributes:    selector.IdentifyingAttributes,
		NonIdentifyingAttributes: selector.NonIdentifyingAttributes,
	}
}

// buildFilter builds a MongoDB filter from a list of conditions.
func buildFilter(conditions []bson.M) bson.M {
	switch len(conditions) {
	case 0:
		return bson.M{}
	case 1:
		return conditions[0]
	default:
		return bson.M{"$and": conditions}
	}
}

// SearchAgents implements port.AgentPersistencePort.
func (a *AgentRepository) SearchAgents(
	ctx context.Context,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	if options == nil {
		//exhaustruct:ignore
		options = &model.ListOptions{}
	}

	// Since instanceUID is stored as bson.Binary, we need to fetch all agents
	// and filter them in memory. For better performance, we should consider:
	// 1. Adding a string representation of instanceUID in the entity
	// 2. Using a different search strategy
	// For now, we'll fetch and filter in memory with pagination
	fetchLimit := int64(100)
	if options.Limit > 0 {
		fetchLimit = options.Limit
	}

	listOptions := &model.ListOptions{
		Limit:    fetchLimit * 10, // Fetch more to account for filtering
		Continue: options.Continue,
	}

	resp, err := a.ListAgents(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents for search: %w", err)
	}

	// Filter agents by instanceUID string match (case-insensitive)
	var filtered []*model.Agent
	queryLower := strings.ToLower(query)
	for _, agent := range resp.Items {
		instanceUIDStr := strings.ToLower(agent.Metadata.InstanceUID.String())
		if strings.Contains(instanceUIDStr, queryLower) {
			filtered = append(filtered, agent)
			if int64(len(filtered)) >= fetchLimit {
				break
			}
		}
	}

	return &model.ListResponse[*model.Agent]{
		Items:              filtered,
		Continue:           resp.Continue,
		RemainingItemCount: 0, // Approximate, since we're filtering in memory
	}, nil
}
