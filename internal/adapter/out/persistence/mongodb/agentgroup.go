package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.AgentGroupPersistencePort = (*AgentGroupMongoAdapter)(nil)

const (
	agentGroupCollectionName = "agentgroups"
)

// AgentGroupMongoAdapter is a struct that implements the AgentGroupPersistencePort interface.
type AgentGroupMongoAdapter struct {
	collection      *mongo.Collection
	agentCollection *mongo.Collection
	common          commonEntityAdapter[entity.AgentGroup, string]
}

// NewAgentGroupRepository creates a new instance of AgentGroupMongoAdapter.
func NewAgentGroupRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *AgentGroupMongoAdapter {
	collection := mongoDatabase.Collection(agentGroupCollectionName)
	agentCollection := mongoDatabase.Collection(agentCollectionName)
	keyFunc := func(en *entity.AgentGroup) string {
		return en.Name
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &AgentGroupMongoAdapter{
		collection:      collection,
		agentCollection: agentCollection,
		common: newCommonAdapter(
			logger,
			collection,
			entity.AgentGroupKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetAgentGroup implements port.AgentGroupPersistencePort.
func (a *AgentGroupMongoAdapter) GetAgentGroup(
	ctx context.Context, name string,
) (*model.AgentGroup, error) {
	entity, err := a.common.get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get agent group: %w", err)
	}

	agentGroupStatistics, err := a.getAgentGroupStatistics(ctx, entity)
	if err != nil {
		return nil, fmt.Errorf("get agent group statistics: %w", err)
	}

	// Convert entity to domain model
	domainModel := entity.ToDomain(agentGroupStatistics)

	return domainModel, nil
}

// ListAgentGroups implements port.AgentGroupPersistencePort.
func (a *AgentGroupMongoAdapter) ListAgentGroups(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*model.AgentGroup], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	// Convert entities to domain models with statistics
	items := make([]*model.AgentGroup, 0, len(resp.Items))
	for _, item := range resp.Items {
		agentGroupStatistics, err := a.getAgentGroupStatistics(ctx, item)
		if err != nil {
			return nil, fmt.Errorf("get agent group statistics for %s: %w", item.Name, err)
		}

		// Convert entity to domain model
		domainModel := item.ToDomain(agentGroupStatistics)
		items = append(items, domainModel)
	}

	return &model.ListResponse[*model.AgentGroup]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutAgentGroup implements port.AgentGroupPersistencePort.
//
//nolint:godox,revive
func (a *AgentGroupMongoAdapter) PutAgentGroup(
	ctx context.Context, name string, agentGroup *model.AgentGroup,
) error {
	// TODO: name should be used to save the agent group with the given name.
	// ref. https://github.com/minuk-dev/opampcommander/issues/145
	// Because some update operations may change the name of the agent group.
	entity := entity.AgentGroupFromDomain(agentGroup)

	err := a.common.put(ctx, entity)
	if err != nil {
		return err
	}

	return nil
}

//nolint:funlen // Reason: mongodb aggregation pipeline is long.
func (a *AgentGroupMongoAdapter) getAgentGroupStatistics(
	ctx context.Context,
	agentGroupEntity *entity.AgentGroup,
) (*entity.AgentGroupStatistics, error) {
	// Build filter conditions for agents matching this agent group's selector
	selector := agentGroupEntity.Selector
	allConditions := SelectorToMatchConditions(selector)

	// Build match filter
	matchFilter := buildFilter(allConditions)

	// MongoDB aggregation pipeline to calculate agent statistics
	pipeline := []bson.M{
		// Match agents that belong to this agent group
		{"$match": matchFilter},

		// Add computed fields for agent conditions
		{
			"$addFields": bson.M{
				"isConnected": bson.M{
					"$anyElementTrue": bson.M{
						"$map": bson.M{
							"input": "$status.conditions",
							"as":    "condition",
							"in": bson.M{
								"$and": []bson.M{
									{"$eq": []any{"$$condition.type", "Connected"}},
									{"$eq": []any{"$$condition.status", "True"}},
								},
							},
						},
					},
				},
				"isHealthy": bson.M{
					"$anyElementTrue": bson.M{
						"$map": bson.M{
							"input": "$status.conditions",
							"as":    "condition",
							"in": bson.M{
								"$and": []bson.M{
									{"$eq": []any{"$$condition.type", "Healthy"}},
									{"$eq": []any{"$$condition.status", "True"}},
								},
							},
						},
					},
				},
			},
		},

		// Group and count by different conditions
		{
			"$group": bson.M{
				"_id":       nil,
				"numAgents": bson.M{"$sum": 1},
				"numConnectedAgents": bson.M{
					"$sum": bson.M{
						"$cond": []any{"$isConnected", 1, 0},
					},
				},
				"numHealthyAgents": bson.M{
					"$sum": bson.M{
						"$cond": []any{
							bson.M{"$and": []any{"$isConnected", "$isHealthy"}}, 1, 0,
						},
					},
				},
				"numUnhealthyAgents": bson.M{
					"$sum": bson.M{
						"$cond": []any{
							bson.M{"$and": []any{"$isConnected", bson.M{"$not": "$isHealthy"}}}, 1, 0,
						},
					},
				},
				"numNotConnectedAgents": bson.M{
					"$sum": bson.M{
						"$cond": []any{bson.M{"$not": "$isConnected"}, 1, 0},
					},
				},
			},
		},
	}

	cursor, err := a.agentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate agent statistics: %w", err)
	}

	defer func() {
		closeErr := cursor.Close(ctx)
		if closeErr != nil {
			a.common.logger.Warn("failed to close mongodb cursor", slog.String("error", closeErr.Error()))
		}
	}()

	var result struct {
		NumAgents             int64 `bson:"numAgents"`
		NumConnectedAgents    int64 `bson:"numConnectedAgents"`
		NumHealthyAgents      int64 `bson:"numHealthyAgents"`
		NumUnhealthyAgents    int64 `bson:"numUnhealthyAgents"`
		NumNotConnectedAgents int64 `bson:"numNotConnectedAgents"`
	}

	if cursor.Next(ctx) {
		err := cursor.Decode(&result)
		if err != nil {
			return nil, fmt.Errorf("failed to decode statistics result: %w", err)
		}
	}

	return &entity.AgentGroupStatistics{
		NumAgents:             result.NumAgents,
		NumConnectedAgents:    result.NumConnectedAgents,
		NumHealthyAgents:      result.NumHealthyAgents,
		NumUnhealthyAgents:    result.NumUnhealthyAgents,
		NumNotConnectedAgents: result.NumNotConnectedAgents,
	}, nil
}
