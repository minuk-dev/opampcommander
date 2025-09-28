package apiserver

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"
	metricapi "go.opentelemetry.io/otel/metric"
	otelpropagation "go.opentelemetry.io/otel/propagation"
	traceapi "go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	experimental "google.golang.org/grpc/experimental/opentelemetry"
	"google.golang.org/grpc/stats/opentelemetry"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// Controller is an interface that defines the methods for handling HTTP requests.
type Controller interface {
	RoutesInfo() gin.RoutesInfo
}

// NewEtcdClient creates a new etcd client with the given settings.
func NewEtcdClient(
	settings *config.ServerSettings,
	meterProvider metricapi.MeterProvider,
	traceProvider traceapi.TracerProvider,
	lifecycle fx.Lifecycle,
) (*clientv3.Client, error) {
	observabilityDialOpt := opentelemetry.DialOption(opentelemetry.Options{
		MetricsOptions: opentelemetry.MetricsOptions{
			MeterProvider:         meterProvider,
			Metrics:               opentelemetry.DefaultMetrics(),
			MethodAttributeFilter: nil,
			OptionalLabels:        nil,
		},
		TraceOptions: experimental.TraceOptions{
			TracerProvider:    traceProvider,
			TextMapPropagator: otelpropagation.TraceContext{},
		},
	})
	//exhaustruct:ignore
	etcdConfig := clientv3.Config{
		Endpoints: settings.DatabaseSettings.Endpoints,
		DialOptions: []grpc.DialOption{
			observabilityDialOpt,
		},
	}

	etcdClient, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, fmt.Errorf("etcd client init failed: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error { return nil },
		OnStop: func(_ context.Context) error {
			err := etcdClient.Close()
			if err != nil {
				return fmt.Errorf("failed to close etcd client: %w", err)
			}

			return nil
		},
	})

	return etcdClient, nil
}
