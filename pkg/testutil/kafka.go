package testutil

import (
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	kafkaTestContainer "github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	kafkaImage        = "confluentinc/cp-kafka:7.5.0"
	kafkaReadyRetries = 60
	kafkaTimeout      = 10 * time.Second
)

// Kafka wraps a testcontainer running a Kafka broker.
type Kafka struct {
	*Base
	testcontainers.Container

	Broker string
}

// StartKafka starts a Kafka container and waits until it is ready to accept connections.
func (b *Base) StartKafka() *Kafka {
	b.t.Helper()

	container, err := kafkaTestContainer.Run(
		b.t.Context(),
		kafkaImage,
		kafkaTestContainer.WithClusterID("test-cluster-id"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("9093/tcp")),
	)
	require.NoError(b.t, err)

	brokers, err := container.Brokers(b.t.Context())
	require.NoError(b.t, err)
	require.NotEmpty(b.t, brokers, "Kafka brokers should not be empty")

	broker := brokers[0]
	b.t.Logf("Kafka started at: %s", broker)

	waitForKafkaReady(b, broker)

	return &Kafka{
		Base:      b,
		Container: container,
		Broker:    broker,
	}
}

//nolint:nestif
func waitForKafkaReady(base *Base, broker string) {
	base.t.Helper()

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_6_0_0
	cfg.Metadata.Timeout = kafkaTimeout
	cfg.Metadata.Retry.Max = 10
	cfg.Metadata.Retry.Backoff = 1 * time.Second
	cfg.Net.DialTimeout = kafkaTimeout
	cfg.Net.ReadTimeout = kafkaTimeout
	cfg.Net.WriteTimeout = kafkaTimeout
	cfg.Admin.Timeout = kafkaTimeout

	for attempt := range kafkaReadyRetries {
		saramaClient, err := sarama.NewClient([]string{broker}, cfg)
		if err == nil {
			bs := saramaClient.Brokers()

			if len(bs) > 0 {
				err = bs[0].Open(cfg)
				if err == nil {
					connected, connErr := bs[0].Connected()

					if connErr == nil && connected {
						base.t.Logf("Kafka is ready after %d retries", attempt+1)
						saramaClient.Close() //nolint:errcheck,gosec

						return
					}

					if connErr != nil {
						base.t.Logf("Kafka broker connection check failed: %v", connErr)
					}
				} else {
					base.t.Logf("Kafka broker open failed: %v", err)
				}
			}

			saramaClient.Close() //nolint:errcheck,gosec
		} else {
			base.t.Logf("Kafka client creation attempt %d failed: %v", attempt+1, err)
		}

		time.Sleep(1 * time.Second)
	}

	base.t.Fatal("Kafka did not become ready in time")
}
