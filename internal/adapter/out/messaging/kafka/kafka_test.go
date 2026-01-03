package kafka_test

import (
	"context"
	"log/slog"
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
	kafkaContainer, broker := startKafkaContainer(t, ctx)

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

	receivedMessages := make(chan *serverevent.Message, 10)
	go consumeMessages(t, ctx, consumer, receivedMessages)

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
	select {
	case received := <-receivedMessages:
		assert.Equal(t, targetServer, received.Target)
		assert.Equal(t, serverevent.MessageTypeSendServerToAgent, received.Type)
		assert.NotNil(t, received.Payload.MessageForServerToAgent)
		assert.Len(t, received.Payload.TargetAgentInstanceUIDs, 1)
		assert.Equal(t, agentUID, received.Payload.TargetAgentInstanceUIDs[0])
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestEventSenderAdapter_SendMultipleMessages(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Given: Kafka is running
	kafkaContainer, broker := startKafkaContainer(t, ctx)

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

	receivedMessages := make(chan *serverevent.Message, 10)
	go consumeMessages(t, ctx, consumer, receivedMessages)

	// When: Send multiple messages
	numMessages := 5
	serverID := "test-server-id"

	for i := range numMessages {
		message := serverevent.Message{
			Target: "target-server-" + string(rune(i)),
			Type:   serverevent.MessageTypeSendServerToAgent,
			Payload: serverevent.MessagePayload{
				MessageForServerToAgent: &serverevent.MessageForServerToAgent{
					TargetAgentInstanceUIDs: []uuid.UUID{uuid.New()},
				},
			},
		}
		err = adapter.SendMessageToServer(ctx, serverID, message)
		require.NoError(t, err)
	}

	// Then: All messages should be received
	receivedCount := 0
	timeout := time.After(20 * time.Second)

	for receivedCount < numMessages {
		select {
		case msg := <-receivedMessages:
			assert.NotNil(t, msg)
			assert.Equal(t, serverevent.MessageTypeSendServerToAgent, msg.Type)

			receivedCount++
		case <-timeout:
			t.Fatalf("Timeout: received %d/%d messages", receivedCount, numMessages)
		}
	}

	assert.Equal(t, numMessages, receivedCount)
}

// Helper functions

func startKafkaContainer(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
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

func consumeMessages(t *testing.T, ctx context.Context, consumer *cekafka.Consumer, messages chan<- *serverevent.Message) {
	t.Helper()

	ceClient, err := cloudevents.NewClient(consumer)
	if err != nil {
		t.Logf("Failed to create cloudevents client: %v", err)

		return
	}

	err = ceClient.StartReceiver(ctx, func(ctx context.Context, event cloudevents.Event) {
		var payload serverevent.MessagePayload

		err := event.DataAs(&payload)
		if err != nil {
			t.Logf("Failed to parse event data: %v", err)

			return
		}

		messageType, err := kafkamodel.MessageTypeFromEventType(event.Type())
		if err != nil {
			t.Logf("Unknown event type: %v", err)

			return
		}

		message := &serverevent.Message{
			Source:  event.Source(),
			Target:  event.Subject(),
			Type:    messageType,
			Payload: payload,
		}

		select {
		case messages <- message:
		case <-ctx.Done():
		}
	})
	if err != nil {
		t.Logf("Failed to start receiver: %v", err)
	}
}
