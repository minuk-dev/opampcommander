package port

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

type ServerEvent struct {
	Source  string
	EventID uuid.UUID
}

// ServerEventhubClient defines the interface to send events to other servers.
type ServerEventhubClient interface {
	Send(ctx context.Context, serverID string) error
}

// EventReceiverClient defines the interface for CloudEvents receiver operations.

// ReceiverFunc defines the function signature for handling received CloudEvents.
type ReceiverFunc func(context.Context, cloudevents.Event) error

// EventPublisher defines the interface for publishing domain events.
type EventPublisher interface {
	// PublishAgentGroupUpdated publishes an event when an agent group is updated.
	PublishAgentGroupUpdated(ctx context.Context, agentGroupName string) error
}

// EventSubscriber defines the interface for subscribing to domain events.
type EventSubscriber interface {
	// SubscribeAgentGroupUpdated subscribes to agent group update events.
	SubscribeAgentGroupUpdated(ctx context.Context) error
}
