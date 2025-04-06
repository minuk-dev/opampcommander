package etcd

import (
	"context"
	"errors"

	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

type CommandEtcdAdapter struct {
	client *clientv3.Client
}

func NewCommandEtcdAdapter(
	client *clientv3.Client,
) *CommandEtcdAdapter {
	return &CommandEtcdAdapter{
		client: client,
	}
}

func (adapter *CommandEtcdAdapter) GetCommand(ctx context.Context, commandID uuid.UUID) (*model.Command, error) {
	return nil, errors.New("not implemented")
}

func (adapter *CommandEtcdAdapter) GetCommandByInstanceUID(ctx context.Context, instanceUID uuid.UUID) (*model.Command, error) {
	return nil, errors.New("not implemented")
}

func (adapter *CommandEtcdAdapter) SaveCommand(ctx context.Context, command *model.Command) error {
	return errors.New("not implemented")
}
