package apiserver

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// Controller is an interface that defines the methods for handling HTTP requests.
type Controller interface {
	RoutesInfo() gin.RoutesInfo
}

// NewEtcdClient creates a new etcd client with the given settings.
func NewEtcdClient(settings *config.ServerSettings, lifecycle fx.Lifecycle) (*clientv3.Client, error) {
	//exhaustruct:ignore
	etcdConfig := clientv3.Config{
		Endpoints: settings.DatabaseSettings.Endpoints,
	}

	etcdClient, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, fmt.Errorf("etcd client init failed: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error { return nil },
		OnStop: func(_ context.Context) error {
			if err := etcdClient.Close(); err != nil {
				return fmt.Errorf("failed to close etcd client: %w", err)
			}

			return nil
		},
	})

	return etcdClient, nil
}
