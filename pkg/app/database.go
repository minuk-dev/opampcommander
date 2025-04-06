package app

import (
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func NewEtcdClient(settings *ServerSettings) (*clientv3.Client, error) {
	etcdConfig := clientv3.Config{
		Endpoints: settings.EtcdHosts,
	}

	etcdClient, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, fmt.Errorf("etcd client init failed: %w", err)
	}

	return etcdClient, nil
}
