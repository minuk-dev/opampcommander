package port

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
)

// AgentPersistencePort is an interface that defines the methods for agent persistence.
type AgentPersistencePort interface {
	// GetAgent retrieves an agent by its instance UID.
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	// GetAgentByID retrieves an agent by its ID.
	PutAgent(ctx context.Context, agent *model.Agent) error
	// ListAgents retrieves a list of agents with pagination options.
	ListAgents(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Agent], error)
	// ListAgentsBySelector retrieves a list of agents matching the given selector with pagination options.
	ListAgentsBySelector(
		ctx context.Context,
		selector model.AgentSelector,
		options *model.ListOptions,
	) (*model.ListResponse[*model.Agent], error)
}

// ServerMessageType represents a message sent to a server.
type ServerMessageType string

// String returns the string representation of the ServerMessageType.
func (s ServerMessageType) String() string {
	return string(s)
}

const (
	// ServerMessageTypeSendServerToAgent is a message type for sending ServerToAgent messages for specific agents.
	ServerMessageTypeSendServerToAgent ServerMessageType = "SendServerToAgent"
)

// ServerMessage represents a message sent between servers.
//
//nolint:embeddedstructfieldcheck // for readability
type ServerMessage struct {
	// Source is the identifier of the message sender of server.
	Source string
	// Target is the identifier of the message recipient agent.
	Target string
	// Type is the type of the message.
	Type ServerMessageType
	// When Type is ServerMessageTypeSendServerToAgent, Payload is ServerToAgentMessage
	*ServerMessageForServerToAgent
}

// ServerMessageForServerToAgent represents a message sent from the server to an agent.
// It's encoded as json in the CloudEvent data field.
type ServerMessageForServerToAgent struct {
	// TargetAgentInstanceUIDs is the list of target agent instance UIDs.
	// Do not send details message, the target server should fetch the details from the database
	// because the message can be delayed or missed.
	// All servers should check all agents status periodically to handle such cases.
	TargetAgentInstanceUIDs []uuid.UUID `json:"targetAgentInstanceUids"`
}

// ServerEventSenderPort is an interface that defines the methods for sending events to servers.
type ServerEventSenderPort interface {
	// SendMessageToServer sends a message to the specified server.
	SendMessageToServer(ctx context.Context, serverID string, message ServerMessage) error
}

// AgentGroupPersistencePort is an interface that defines the methods for agent group persistence.
type AgentGroupPersistencePort interface {
	// GetAgentGroup retrieves an agent group by its ID.
	GetAgentGroup(ctx context.Context, name string) (*agentgroup.AgentGroup, error)
	// PutAgentGroup saves the agent group.
	PutAgentGroup(ctx context.Context, name string, agentGroup *agentgroup.AgentGroup) error
	// ListAgentGroups retrieves a list of agent groups with pagination options.
	ListAgentGroups(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*agentgroup.AgentGroup], error)
}

// CommandPersistencePort is an interface that defines the methods for command persistence.
type CommandPersistencePort interface {
	// GetCommand retrieves a command by its ID.
	GetCommand(ctx context.Context, commandID uuid.UUID) (*model.Command, error)
	// GetCommandByInstanceUID retrieves a command by its instance UID.
	GetCommandByInstanceUID(ctx context.Context, instanceUID uuid.UUID) (*model.Command, error)
	// SaveCommand saves the command.
	SaveCommand(ctx context.Context, command *model.Command) error
}

// ServerPersistencePort is an interface that defines the methods for server persistence.
type ServerPersistencePort interface {
	// GetServer retrieves a server by its ID.
	GetServer(ctx context.Context, id string) (*model.Server, error)
	// PutServer saves or updates a server.
	PutServer(ctx context.Context, server *model.Server) error
	// ListServers retrieves a list of all servers.
	ListServers(ctx context.Context) ([]*model.Server, error)
}

var (
	// ErrResourceNotExist is an error that indicates that the resource does not exist.
	ErrResourceNotExist = errors.New("resource does not exist")
	// ErrMultipleResourceExist is an error that indicates that multiple resources exist.
	ErrMultipleResourceExist = errors.New("multiple resources exist")
)
