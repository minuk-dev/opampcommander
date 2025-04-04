package etcd

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	clientv3 "go.etcd.io/etcd/client/v3"
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
	return nil, fmt.Errorf("not implemented")
}

func (adapter *CommandEtcdAdapter) GetCommandByInstanceUID(ctx context.Context, instanceUID uuid.UUID) (*model.Command, error) {
	return nil, fmt.Errorf("not implemented")
}

func (adapter *CommandEtcdAdapter) SaveCommand(ctx context.Context, command *model.Command) error {
	return fmt.Errorf("not implemented")
}
