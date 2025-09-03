package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/samber/lo"
	clientv3 "go.etcd.io/etcd/client/v3"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

type ToEntityFunc[Domain any] func(domain *Domain) (Entity[Domain], error)

type KeyFunc[Domain any] func(domain *Domain) string

type Entity[Domain any] interface {
	ToDomain() *Domain
}

type commonAdapter[Domain any] struct {
	client       *clientv3.Client
	logger       *slog.Logger
	ToEntityFunc ToEntityFunc[Domain]
	KeyPrefix    string
	KeyFunc      KeyFunc[Domain]
}

func newCommonAdapter[Domain any](
	client *clientv3.Client,
	logger *slog.Logger,
	toEntityFunc ToEntityFunc[Domain],
	keyPrefix string,
	keyFunc KeyFunc[Domain],
) commonAdapter[Domain] {
	return commonAdapter[Domain]{
		client:       client,
		logger:       logger,
		ToEntityFunc: toEntityFunc,
		KeyPrefix:    keyPrefix,
		KeyFunc:      keyFunc,
	}
}

func (a *commonAdapter[Domain]) get(ctx context.Context, keyWithoutPrefix string) (*Domain, error) {
	key := a.KeyPrefix + keyWithoutPrefix

	getResponse, err := a.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource from etcd: %w", err)
	}

	if getResponse.Count == 0 {
		return nil, domainport.ErrResourceNotExist
	}

	if getResponse.Count > 1 {
		// it should not happen, but if it does, we return an error
		// it's untestable because we always put a single resource with a unique key
		return nil, domainport.ErrMultipleResourceExist
	}

	var entity Entity[Domain]

	err = json.Unmarshal(getResponse.Kvs[0].Value, &entity)
	if err != nil {
		return nil, fmt.Errorf("failed to decode resource from received data: %w", err)
	}

	return entity.ToDomain(), nil
}

func (a *commonAdapter[Domain]) list(ctx context.Context, options *domainmodel.ListOptions) (*domainmodel.ListResponse[*Domain], error) {
	if options == nil {
		options = &domainmodel.ListOptions{
			Limit:    0,  // 0 means no limit
			Continue: "", // empty continue token means start from the beginning
		}
	}

	startKey := a.KeyPrefix + options.Continue

	getResponse, err := a.client.Get(
		ctx,
		startKey,
		clientv3.WithLimit(options.Limit),
		clientv3.WithRange(a.KeyPrefix+"\xFF"), // Use a range to get all keys under "agents/"
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources from etcd: %w", err)
	}

	domains := make([]*Domain, 0, getResponse.Count)

	for _, kv := range getResponse.Kvs {
		var entity Entity[Domain]

		err = json.Unmarshal(kv.Value, &entity)
		if err != nil {
			return nil, fmt.Errorf("failed to decode resource from received data: %w", err)
		}

		domain := entity.ToDomain()
		domains = append(domains, domain)
	}
	// Use a null byte to ensure the next key is lexicographically greater
	var continueKey string
	if len(domains) > 0 {
		continueKey = a.KeyFunc(lo.LastOrEmpty(domains)) + "\x00"
	}

	return &domainmodel.ListResponse[*Domain]{
		RemainingItemCount: getResponse.Count - int64(len(domains)),
		Continue:           continueKey,
		Items:              domains,
	}, nil
}

func (a *commonAdapter[Domain]) put(ctx context.Context, domain *Domain) error {
	key := a.KeyPrefix + a.KeyFunc(domain)

	entity, err := a.ToEntityFunc(domain)
	if err != nil {
		return fmt.Errorf("failed to convert domain to entity: %w", err)
	}

	data, err := json.Marshal(entity)
	if err != nil {
		return fmt.Errorf("failed to encode entity to JSON: %w", err)
	}

	_, err = a.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to put resource to etcd: %w", err)
	}

	return nil
}
