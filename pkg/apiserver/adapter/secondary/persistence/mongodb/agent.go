package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

const (
	resourceVersionFieldName = "metadata.resourceVersion"
)

var (
	_ agentport.AgentPersistencePort = (*AgentRepository)(nil)

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

// GetAgent implements agentport.AgentPersistencePort.
func (a *AgentRepository) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	entity, err := a.common.get(ctx, instanceUID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent from persistence: %w", err)
	}

	return entity.ToDomain(), nil
}

// ListAgents implements agentport.AgentPersistencePort.
func (a *AgentRepository) ListAgents(
	ctx context.Context,
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	conditions := []bson.M{{"metadata.namespace": sanitizeResourceName(namespace)}}

	if options != nil {
		if options.ConnectedOnly {
			conditions = append(conditions, connectedMatchFilter())
		}
		// Each attribute condition is a separate $elemMatch on the same field, so
		// they must be combined with $and (via buildFilter) rather than flattened
		// into one map, which would drop all but the last.
		conditions = append(conditions,
			IdentifyingAttributesSelectorToMatchConditions(options.IdentifyingAttributes)...)
		conditions = append(conditions,
			NonIdentifyingAttributesSelectorToMatchConditions(options.NonIdentifyingAttributes)...)
	}

	resp, err := a.common.listWithFilter(ctx, options, buildFilter(conditions))
	if err != nil {
		return nil, fmt.Errorf("failed to list agents from persistence: %w", err)
	}

	return &model.ListResponse[*agentmodel.Agent]{
		Items: lo.Map(resp.Items, func(item *entity.Agent, _ int) *agentmodel.Agent {
			return item.ToDomain()
		}),
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutAgent implements agentport.AgentPersistencePort.
//
// PutAgent is an optimistic-concurrency write: it only succeeds when the stored
// document's resourceVersion still equals the version the in-memory agent was
// loaded with (agent.Metadata.ResourceVersion). On success the stored version is
// incremented and the increment is written back onto the passed agent, so the
// caller — and any cache that clones it afterwards — holds the new version. A
// concurrent writer that already advanced the version (another HA node, the
// reconcile loop, a racing API call) makes this return [model.ErrConflict] instead
// of silently clobbering that writer's change.
//
// The create case (expected version 0) relies on the unique index on
// metadata.instanceUid: if the agent already exists, the version filter misses and
// the upsert attempts an insert that the unique index rejects as a duplicate key,
// which is surfaced as a conflict rather than a duplicate document.
func (a *AgentRepository) PutAgent(ctx context.Context, agent *agentmodel.Agent) error {
	expected := agent.Metadata.ResourceVersion
	next := expected + 1

	doc := entity.AgentFromDomain(agent)
	doc.Metadata.ResourceVersion = next

	filter := bson.M{
		entity.AgentKeyFieldName: a.common.KeyQueryFunc(agent.Metadata.InstanceUID),
		resourceVersionFieldName: expected,
	}

	result, err := a.collection.ReplaceOne(ctx, filter, doc, options.Replace().SetUpsert(expected == 0))
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%w: agent %s was created concurrently", model.ErrConflict, agent.Metadata.InstanceUID)
		}

		return fmt.Errorf("failed to put agent to persistence: %w", err)
	}

	// No matched (and not freshly upserted) document means the version filter did not
	// find the expected version — another writer advanced it (or deleted the agent).
	if result.MatchedCount == 0 && result.UpsertedCount == 0 {
		return fmt.Errorf("%w: agent %s was modified concurrently", model.ErrConflict, agent.Metadata.InstanceUID)
	}

	agent.Metadata.ResourceVersion = next

	return nil
}

// UpdateAgentLiveness implements agentport.AgentPersistencePort.
//
// A targeted $set of the four liveness fields, with no resource-version filter and
// no version bump: liveness carries no optimistic-concurrency meaning, and bumping
// the version on this cadence would make routine heartbeats invalidate concurrent
// API writes. Last write wins, which is the right semantic for a timestamp whose
// only job is to be recent.
func (a *AgentRepository) UpdateAgentLiveness(
	ctx context.Context,
	liveness *agentmodel.AgentLiveness,
) error {
	filter := bson.M{entity.AgentKeyFieldName: a.common.KeyQueryFunc(liveness.InstanceUID)}
	update := bson.M{"$set": bson.M{
		"status.connected":          liveness.Connected,
		"status.connectionType":     liveness.ConnectionType.String(),
		"status.sequenceNum":        liveness.SequenceNum,
		"status.lastCommunicatedAt": bson.NewDateTimeFromTime(liveness.LastReportedAt),
		"status.lastCommunicatedTo": liveness.LastReportedTo,
	}}

	result, err := a.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update agent liveness in persistence: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("%w: agent %s", model.ErrResourceNotExist, liveness.InstanceUID)
	}

	return nil
}

// DeleteAgent implements agentport.AgentPersistencePort.
func (a *AgentRepository) DeleteAgent(ctx context.Context, instanceUID uuid.UUID) error {
	err := a.common.deleteOne(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to delete agent from persistence: %w", err)
	}

	return nil
}

// ListAgentsBySelector implements agentport.AgentPersistencePort.
func (a *AgentRepository) ListAgentsBySelector(
	ctx context.Context,
	selector agentmodel.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	if options == nil {
		//exhaustruct:ignore
		options = &model.ListOptions{}
	}

	continueTokenObjectID, err := bson.ObjectIDFromHex(options.Continue)
	if err != nil && options.Continue != "" {
		return nil, fmt.Errorf("invalid continue token: %w", err)
	}

	conditions := SelectorToMatchConditions(AgentSelectorToEntity(selector))
	if options.ConnectedOnly {
		conditions = append(conditions, connectedMatchFilter())
	}

	prefix := mongo.Pipeline{bson.D{{Key: "$match", Value: buildFilter(conditions)}}}

	entities, continueToken, remaining, err := aggregateListPage[entity.Agent](
		ctx, a.logger, a.collection, prefix, continueTokenObjectID, options.Limit,
	)
	if err != nil {
		return nil, err
	}

	return &model.ListResponse[*agentmodel.Agent]{
		Items: lo.Map(entities, func(item *entity.Agent, _ int) *agentmodel.Agent {
			return item.ToDomain()
		}),
		Continue:           continueToken,
		RemainingItemCount: remaining,
	}, nil
}

// AgentSelectorToEntity converts a domain AgentSelector to a persistence entity AgentSelector.
func AgentSelectorToEntity(selector agentmodel.AgentSelector) entity.AgentSelector {
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

// SearchAgents implements agentport.AgentPersistencePort.
func (a *AgentRepository) SearchAgents(
	ctx context.Context,
	namespace string,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
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
		return &model.ListResponse[*agentmodel.Agent]{
			Items:              []*agentmodel.Agent{},
			Continue:           "",
			RemainingItemCount: 0,
		}, nil
	}

	continueTokenObjectID, err := bson.ObjectIDFromHex(options.Continue)
	if err != nil && options.Continue != "" {
		return nil, fmt.Errorf("invalid continue token: %w", err)
	}

	prefix := mongo.Pipeline{
		bson.D{{Key: "$match", Value: buildFilter(a.buildSearchConditions(namespace, query, options))}},
	}

	entities, continueToken, remaining, err := aggregateListPage[entity.Agent](
		ctx, a.logger, a.collection, prefix, continueTokenObjectID, options.Limit,
	)
	if err != nil {
		return nil, err
	}

	return &model.ListResponse[*agentmodel.Agent]{
		Items: lo.Map(entities, func(item *entity.Agent, _ int) *agentmodel.Agent {
			return item.ToDomain()
		}),
		Continue:           continueToken,
		RemainingItemCount: remaining,
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

// buildSearchConditions builds the match conditions for a non-empty search query. The
// continue-token condition is applied by [aggregateListPage], not here.
func (a *AgentRepository) buildSearchConditions(
	namespace string,
	query string,
	options *model.ListOptions,
) []bson.M {
	// Prefix-match instanceUidString with a parameterized range scan instead of a
	// user-built $regex: instanceUidString is always a lower-cased UUID, so we
	// lower-case the query (preserving the previous case-insensitive behaviour) and
	// match [prefix, prefixUpperBound). This avoids feeding user input into a regex
	// (no ReDoS / regex injection) and lets the index serve the query with a range
	// scan rather than a regex scan.
	lowerQuery := strings.ToLower(query)

	prefixRange := bson.M{"$gte": lowerQuery}
	if upper, ok := prefixUpperBound(lowerQuery); ok {
		prefixRange["$lt"] = upper
	}

	conditions := []bson.M{
		{"metadata.namespace": namespace},
		{"metadata.instanceUidString": prefixRange},
	}

	if options.ConnectedOnly {
		conditions = append(conditions, connectedMatchFilter())
	}

	return conditions
}

// ensureIndexes creates necessary indexes for the agent collection.
func (a *AgentRepository) ensureIndexes(ctx context.Context) {
	//exhaustruct:ignore
	searchIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "metadata.instanceUidString", Value: 1},
		},
	}

	_, err := a.collection.Indexes().CreateOne(ctx, searchIndex)
	if err != nil {
		a.logger.Warn("failed to create index for instanceUidString", slog.String("error", err.Error()))
	}

	// The unique index on metadata.instanceUid — which PutAgent's optimistic-concurrency
	// create path relies on — is owned by the centralized EnsureSchema (mongodb.go) so it
	// is not declared twice with conflicting options.
}

// prefixUpperBound returns the exclusive upper bound for a prefix range scan: the
// smallest string strictly greater than every string starting with prefix, using
// MongoDB's default binary (byte-wise) string ordering. It increments the last
// byte that can be incremented and drops the rest. ok is false when no finite
// bound exists (prefix is empty or all 0xFF bytes), in which case the caller
// should omit the upper bound and rely on $gte alone.
func prefixUpperBound(prefix string) (string, bool) {
	const maxByte = 0xFF

	bytes := []byte(prefix)
	for i := len(bytes) - 1; i >= 0; i-- {
		if bytes[i] < maxByte {
			bytes[i]++

			return string(bytes[:i+1]), true
		}
	}

	return "", false
}
