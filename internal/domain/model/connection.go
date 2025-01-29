package model

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
)

type Connection interface {
	ID() uuid.UUID
	SendServerToAgent(ctx context.Context, serverToAgent *protobufs.ServerToAgent) error
}
