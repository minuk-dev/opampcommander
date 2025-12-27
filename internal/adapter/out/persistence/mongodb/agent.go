package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sync"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var (
	_ domainport.AgentPersistencePort = (*AgentRepository)(nil)

	// ErrQueryTooLong is returned when the search query exceeds the maximum length.
	ErrQueryTooLong = errors.New("query too long: maximum length is 100 characters")
	// ErrQueryTooShort is returned when the search query is too short.
	ErrQueryTooShort = errors.New("query too short: minimum length is 1 character")
)

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

	repo := &AgentRepository{
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

	// Create index for instanceUidString for efficient searching
	repo.ensureIndexes(context.Background())

	return repo
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

	// Validate query
	err := validateSearchQuery(query)
	if err != nil {
		return nil, err
	}

	// Return empty result for empty query
	if query == "" {
		return &model.ListResponse[*model.Agent]{
			Items:              []*model.Agent{},
			Continue:           "",
			RemainingItemCount: 0,
		}, nil
	}

	// Build search filter
	filter, err := a.buildSearchFilter(query, options)
	if err != nil {
		return nil, err
	}

	// Execute parallel queries
	entities, continueToken, count, err := a.executeSearchQueries(ctx, filter, options)
	if err != nil {
		return nil, err
	}

	// Convert to domain models
	return &model.ListResponse[*model.Agent]{
		Items: lo.Map(entities, func(item *entity.Agent, _ int) *model.Agent {
			return item.ToDomain()
		}),
		Continue:           continueToken,
		RemainingItemCount: count - int64(len(entities)),
	}, nil
}

func validateSearchQuery(query string) error {
	if query == "" {
		return nil // Empty query is valid, handled separately
	}

	const (
		maxQueryLength = 100
		minQueryLength = 1
	)

	if len(query) > maxQueryLength {
		return ErrQueryTooLong
	}

	if len(query) < minQueryLength {
		return ErrQueryTooShort
	}

	return nil
}

func (a *AgentRepository) buildSearchFilter(query string, options *model.ListOptions) (bson.M, error) {
	if query == "" {
		return bson.M{}, nil
	}

	continueTokenObjectID, err := bson.ObjectIDFromHex(options.Continue)
	if err != nil && options.Continue != "" {
		return nil, fmt.Errorf("invalid continue token: %w", err)
	}

	// Escape regex metacharacters to prevent regex injection
	safeQuery := escapeRegexLiteral(query)

	conditions := []bson.M{
		{"metadata.instanceUidString": bson.M{"$regex": "^" + safeQuery, "$options": "i"}},
	}

	// Add continue token condition if present
	continueTokenFilter := withContinueToken(continueTokenObjectID)
	if continueTokenFilter != nil {
		conditions = append(conditions, continueTokenFilter)
	}

	return buildFilter(conditions), nil
}

func (a *AgentRepository) executeSearchQueries(
	ctx context.Context,
	filter bson.M,
	options *model.ListOptions,
) ([]*entity.Agent, string, int64, error) {
	var (
		entities      []*entity.Agent
		continueToken string
		count         int64
		findErr       error
		countErr      error
	)

	var queryWg sync.WaitGroup

	queryWg.Go(func() {
		entities, continueToken, findErr = a.findAgents(ctx, filter, options)
	})

	queryWg.Go(func() {
		count, countErr = a.countAgents(ctx, filter)
	})

	queryWg.Wait()

	if findErr != nil || countErr != nil {
		return nil, "", 0, fmt.Errorf("search operation failed: %w %w", findErr, countErr)
	}

	return entities, continueToken, count, nil
}

func (a *AgentRepository) findAgents(
	ctx context.Context,
	filter bson.M,
	options *model.ListOptions,
) ([]*entity.Agent, string, error) {
	cursor, err := a.collection.Find(ctx, filter, withLimit(options.Limit))
	if err != nil {
		return nil, "", fmt.Errorf("failed to search agents from mongodb: %w", err)
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
		return nil, "", fmt.Errorf("failed to decode search agents from mongodb: %w", err)
	}

	continueToken, err := getContinueTokenFromEntities(entities)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get continue token from entities: %w", err)
	}

	return entities, continueToken, nil
}

func (a *AgentRepository) countAgents(ctx context.Context, filter bson.M) (int64, error) {
	count, err := a.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count search agents in mongodb: %w", err)
	}

	return count, nil
}

// ensureIndexes creates necessary indexes for the agent collection.
func (a *AgentRepository) ensureIndexes(ctx context.Context) {
	//exhaustruct:ignore
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "metadata.instanceUidString", Value: 1},
		},
	}

	_, err := a.collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		a.logger.Warn("failed to create index for instanceUidString", slog.String("error", err.Error()))
	}
}

// escapeRegexLiteral escapes all regular expression metacharacters in the input
// so that it is treated as a literal string within a regex pattern.
func escapeRegexLiteral(query string) string {
	return regexp.QuoteMeta(query)
}
