package kafka_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	cekafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	kafkaTestContainer "github.com/testcontainers/testcontainers-go/modules/kafka"

	kafkamodel "github.com/minuk-dev/opampcommander/internal/adapter/common/kafka"
	outkafka "github.com/minuk-dev/opampcommander/internal/adapter/out/messaging/kafka"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestEventSenderAdapter_SendMessageToServer(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Given: Kafka is running
	kafkaContainer, broker := startKafkaContainer(ctx, t)

	defer func() { _ = kafkaContainer.Terminate(ctx) }()

	topic := "test-topic-" + uuid.New().String()

	// Given: EventSenderAdapter is created
	sender := createTestSender(t, broker, topic)
	logger := slog.New(slog.NewTextHandler(testutil.TestLogWriter{T: t}, nil))
	adapter, err := outkafka.NewEventSenderAdapter(sender, logger)
	require.NoError(t, err)

	// Given: Consumer to verify messages
	consumer := createTestConsumer(t, broker, topic)

	defer func() { _ = consumer.Close(ctx) }()

	// When: Send message
	serverID := "test-server-id"
	targetServer := "target-server-id"
	agentUID := uuid.New()
	message := serverevent.Message{
		Target: targetServer,
		Type:   serverevent.MessageTypeSendServerToAgent,
		Payload: serverevent.MessagePayload{
			MessageForServerToAgent: &serverevent.MessageForServerToAgent{
				TargetAgentInstanceUIDs: []uuid.UUID{agentUID},
			},
		},
	}

	err = adapter.SendMessageToServer(ctx, serverID, message)
	require.NoError(t, err)

	// Then: Message should be received
	received := consumeMessage(ctx, t, consumer)
	require.NotNil(t, received, "received message should not be nil")
	assert.Equal(t, targetServer, received.Target)
	assert.Equal(t, serverevent.MessageTypeSendServerToAgent, received.Type)
	assert.NotNil(t, received.Payload.MessageForServerToAgent)
	assert.Len(t, received.Payload.TargetAgentInstanceUIDs, 1)
	assert.Equal(t, agentUID, received.Payload.TargetAgentInstanceUIDs[0])
}

// Helper functions.
func startKafkaContainer(ctx context.Context, t *testing.T) (testcontainers.Container, string) {
	t.Helper()

	kafkaContainer, err := kafkaTestContainer.Run(ctx,
		"confluentinc/cp-kafka:7.5.0",
		kafkaTestContainer.WithClusterID("test-cluster-id"),
	)
	require.NoError(t, err)

	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, brokers)

	return kafkaContainer, brokers[0]
}

func createTestSender(t *testing.T, broker, topic string) *cekafka.Sender {
	t.Helper()

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Version = sarama.V2_6_0_0

	sender, err := cekafka.NewSender([]string{broker}, config, topic)
	require.NoError(t, err)

	return sender
}

func createTestConsumer(t *testing.T, broker, topic string) *cekafka.Consumer {
	t.Helper()

	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Version = sarama.V2_6_0_0

	groupID := "test-consumer-group-" + uuid.New().String()
	consumer, err := cekafka.NewConsumer([]string{broker}, config, groupID, topic)
	require.NoError(t, err)

	return consumer
}

// consumeMessage receives a single message from the Kafka consumer.
func consumeMessage(ctx context.Context, t *testing.T, consumer *cekafka.Consumer) *serverevent.Message {
	t.Helper()

	receiveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := cloudevents.NewClient(consumer)
	require.NoError(t, err)

	var msg *serverevent.Message

	var wg sync.WaitGroup
	wg.Go(func() {
		err := client.StartReceiver(receiveCtx, func(_ context.Context, e cloudevents.Event) {
			var payload serverevent.MessagePayload

			err := e.DataAs(&payload)
			require.NoError(t, err)

			messageType, err := kafkamodel.MessageTypeFromEventType(e.Type())
			require.NoError(t, err)

			msg = &serverevent.Message{
				Source:  e.Source(),
				Target:  e.Subject(),
				Type:    messageType,
				Payload: payload,
			}

			cancel() // Stop receiving after first message
		})
		require.NoError(t, err)
	})
	wg.Wait()

	return msg
}
