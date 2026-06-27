package inmemory

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var (
	_ agentport.AgentPersistencePort = (*AgentRepository)(nil)

	// ErrQueryTooLong is returned when the search query exceeds the maximum length.
	ErrQueryTooLong = errors.New("query too long: maximum length is 100 characters")
)

// AgentRepository is the in-memory implementation of [agentport.AgentPersistencePort].
//
// Agents have no soft-delete concept (DeleteAgent is a hard delete), matching
// the MongoDB adapter.
type AgentRepository struct {
	store *store[uuid.UUID, *agentmodel.Agent]
	clock clock.PassiveClock
}

// NewAgentRepository creates a new in-memory AgentRepository.
func NewAgentRepository() *AgentRepository {
	return &AgentRepository{
		store: newStore[uuid.UUID]((*agentmodel.Agent).Clone, nil),
		clock: clock.NewRealClock(),
	}
}

// GetAgent implements agentport.AgentPersistencePort.
func (r *AgentRepository) GetAgent(_ context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	return r.store.get(instanceUID, nil)
}

// PutAgent implements agentport.AgentPersistencePort.
//
// Like the MongoDB adapter, this is an optimistic-concurrency write: it succeeds
// only if the stored agent's ResourceVersion still equals the version the passed
// agent was loaded with, otherwise it returns [port.ErrConflict]. On success the
// version is incremented and written back onto the passed agent.
func (r *AgentRepository) PutAgent(_ context.Context, agent *agentmodel.Agent) error {
	expected := agent.Metadata.ResourceVersion
	next := expected + 1

	toStore := agent.Clone()
	toStore.Metadata.ResourceVersion = next

	err := r.store.casPut(agent.Metadata.InstanceUID, toStore, expected, func(a *agentmodel.Agent) int64 {
		return a.Metadata.ResourceVersion
	})
	if err != nil {
		return err
	}

	agent.Metadata.ResourceVersion = next

	return nil
}

// DeleteAgent implements agentport.AgentPersistencePort.
func (r *AgentRepository) DeleteAgent(_ context.Context, instanceUID uuid.UUID) error {
	return r.store.delete(instanceUID)
}

// ListAgents implements agentport.AgentPersistencePort.
func (r *AgentRepository) ListAgents(
	_ context.Context,
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	connectedOnly := options != nil && options.ConnectedOnly

	var identifyingAttributes, nonIdentifyingAttributes map[string]string
	if options != nil {
		identifyingAttributes = options.IdentifyingAttributes
		nonIdentifyingAttributes = options.NonIdentifyingAttributes
	}

	return r.store.list(options, func(agent *agentmodel.Agent) bool {
		if agent.Metadata.Namespace != namespace {
			return false
		}

		if !matchesAttributes(agent.Metadata.Description.IdentifyingAttributes, identifyingAttributes) {
			return false
		}

		if !matchesAttributes(agent.Metadata.Description.NonIdentifyingAttributes, nonIdentifyingAttributes) {
			return false
		}

		return !connectedOnly || r.isConnected(agent)
	})
}

// ListAgentsBySelector implements agentport.AgentPersistencePort.
func (r *AgentRepository) ListAgentsBySelector(
	_ context.Context,
	selector agentmodel.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	connectedOnly := options != nil && options.ConnectedOnly

	return r.store.list(options, func(agent *agentmodel.Agent) bool {
		if !matchesSelector(agent, selector) {
			return false
		}

		return !connectedOnly || r.isConnected(agent)
	})
}

// SearchAgents implements agentport.AgentPersistencePort.
func (r *AgentRepository) SearchAgents(
	_ context.Context,
	namespace string,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	const maxQueryLength = 100
	if len(query) > maxQueryLength {
		return nil, ErrQueryTooLong
	}

	if query == "" {
		return &model.ListResponse[*agentmodel.Agent]{
			Items:              []*agentmodel.Agent{},
			Continue:           "",
			RemainingItemCount: 0,
		}, nil
	}

	connectedOnly := options != nil && options.ConnectedOnly
	prefix := strings.ToLower(query)

	return r.store.list(options, func(agent *agentmodel.Agent) bool {
		if agent.Metadata.Namespace != namespace {
			return false
		}

		if !strings.HasPrefix(strings.ToLower(agent.Metadata.InstanceUID.String()), prefix) {
			return false
		}

		return !connectedOnly || r.isConnected(agent)
	})
}

// isConnected mirrors the MongoDB connected filter: the explicit Connected flag
// plus heartbeat staleness, evaluated against the repository clock.
func (r *AgentRepository) isConnected(agent *agentmodel.Agent) bool {
	return agent.IsConnectedAt(r.clock.Now(), agentmodel.DefaultConnectionStaleness)
}

// agentsMatchingSelector returns every stored agent (including hard-undeletable
// ones) matching the selector. Used by the agent-group repository to compute
// group statistics.
func (r *AgentRepository) agentsMatchingSelector(selector agentmodel.AgentSelector) []*agentmodel.Agent {
	return r.store.snapshot(false, func(agent *agentmodel.Agent) bool {
		return matchesSelector(agent, selector)
	})
}
