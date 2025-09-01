package etcd

import (
	"context"
	"log/slog"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type ToEntityFunc[Domain any] func(domain Domain) (Entity[Domain], error)

type Entity[Domain any] interface {
	ToDomain() Domain
}

type commonAdapter[Domain any] struct {
	client       *clientv3.Client
	logger       *slog.Logger
	ToEntityFunc ToEntityFunc[Domain]
}

func (a *commonAdapter[Domain]) get(ctx context.Context, key string) (*Domain, error) {
}

func (a *commonAdapter[Domain]) delete(ctx context.Context, key string) error {
}

func (a *commonAdapter[Domain]) list(ctx context.Context, prefix string) ([]Domain, error) {
}

func (a *commonAdapter[Domain]) put(ctx context.Context, key string, domain Domain) error {
	entity, err := a.ToEntityFunc(domain)
	if err != nil {
		return err
	}
}
