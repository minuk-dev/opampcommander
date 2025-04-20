package etcd

import (
	"context"
	"errors"

	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// CommandEtcdAdapter is a struct that implements the CommandPersistencePort interface.
type CommandEtcdAdapter struct {
	client *clientv3.Client
}

// NewCommandEtcdAdapter creates a new instance of CommandEtcdAdapter.
func NewCommandEtcdAdapter(
	client *clientv3.Client,
) *CommandEtcdAdapter {
	return &CommandEtcdAdapter{
		client: client,
	}
}

// ErrNotImplemented is an error that indicates that the method is not implemented.
var ErrNotImplemented = errors.New("not implemented")

// GetCommand retrieves a command by its ID.
func (adapter *CommandEtcdAdapter) GetCommand(context.Context, uuid.UUID) (*model.Command, error) {
	return nil, ErrNotImplemented
}

// GetCommandByInstanceUID retrieves a command by its instance UID.
func (adapter *CommandEtcdAdapter) GetCommandByInstanceUID(context.Context, uuid.UUID) (*model.Command, error) {
	return nil, ErrNotImplemented
}

// SaveCommand saves the command to the persistence layer.
func (adapter *CommandEtcdAdapter) SaveCommand(context.Context, *model.Command) error {
	return ErrNotImplemented
}
