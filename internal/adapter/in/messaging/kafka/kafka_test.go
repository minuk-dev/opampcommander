package kafka_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/IBM/sarama"
	cekafka "github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	kafkaTestContainer "github.com/testcontainers/testcontainers-go/modules/kafka"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/kafka"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
)

func TestKafkaAdapter_SendAndReceive(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping Kafka integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Given: Kafka is running
	kafkaContainer, broker := startKafkaContainer(t)

	defer func() { _ = kafkaContainer.Terminate(ctx) }()

	// Given: Kafka sender and receiver are configured
	topic := "test.opampcommander.events"
	sender := createTestSender(t, broker, topic)
	receiver := createTestReceiver(t, broker, topic)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	adapter, err := kafka.NewEventSenderAdapter(sender, receiver, logger)
	require.NoError(t, err)

	// Given: Receiver is started
	receivedMessages := make(chan *serverevent.Message, 10)

	go func() {
		_ = adapter.StartReceiver(ctx, func(_ context.Context, msg *serverevent.Message) error {
			receivedMessages <- msg

			return nil
		})
	}()

	// Allow receiver to start
	time.Sleep(2 * time.Second)

	// When: Message is sent
	testAgentUID := uuid.New()
	testMessage := serverevent.Message{
		Source: "test-server-1",
		Target: "test-server-2",
		Type:   serverevent.MessageTypeSendServerToAgent,
		Payload: serverevent.MessagePayload{
			MessageForServerToAgent: &serverevent.MessageForServerToAgent{
				TargetAgentInstanceUIDs: []uuid.UUID{testAgentUID},
			},
		},
	}

	err = adapter.SendMessageToServer(ctx, "test-server-1", testMessage)
	require.NoError(t, err)

	// Then: Message should be received
	select {
	case received := <-receivedMessages:
		assert.Equal(t, testMessage.Type, received.Type)
		require.NotNil(t, received.Payload.MessageForServerToAgent)
		assert.Equal(t, testAgentUID, received.Payload.TargetAgentInstanceUIDs[0])
		assert.Contains(t, received.Source, "test-server-1")
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestKafkaAdapter_MultipleMessages(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping Kafka integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	kafkaContainer, broker := startKafkaContainer(t)

	defer func() { _ = kafkaContainer.Terminate(ctx) }()

	topic := "test.opampcommander.multi"
	sender := createTestSender(t, broker, topic)
	receiver := createTestReceiver(t, broker, topic)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	adapter, err := kafka.NewEventSenderAdapter(sender, receiver, logger)
	require.NoError(t, err)

	receivedMessages := make(chan *serverevent.Message, 10)

	go func() {
		_ = adapter.StartReceiver(ctx, func(_ context.Context, msg *serverevent.Message) error {
			receivedMessages <- msg

			return nil
		})
	}()

	time.Sleep(2 * time.Second)

	// When: Multiple messages are sent
	numMessages := 5
	for range numMessages {
		agentUID := uuid.New()
		msg := serverevent.Message{
			Source: "test-server",
			Type:   serverevent.MessageTypeSendServerToAgent,
			Payload: serverevent.MessagePayload{
				MessageForServerToAgent: &serverevent.MessageForServerToAgent{
					TargetAgentInstanceUIDs: []uuid.UUID{agentUID},
				},
			},
		}
		err := adapter.SendMessageToServer(ctx, "test-server", msg)
		require.NoError(t, err)
	}

	// Then: All messages should be received
	receivedCount := 0
	timeout := time.After(15 * time.Second)

	for receivedCount < numMessages {
		select {
		case msg := <-receivedMessages:
			assert.NotNil(t, msg)

			receivedCount++
		case <-timeout:
			t.Fatalf("Timeout: received only %d out of %d messages", receivedCount, numMessages)
		}
	}

	assert.Equal(t, numMessages, receivedCount)
}

// Helper functions

//nolint:ireturn
func startKafkaContainer(t *testing.T) (testcontainers.Container, string) {
	t.Helper()
	ctx := t.Context()

	kafkaContainer, err := kafkaTestContainer.Run(ctx, "confluentinc/cp-kafka:7.5.0")
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

	sender, err := cekafka.NewSender([]string{broker}, config, topic)
	require.NoError(t, err)

	return sender
}

func createTestReceiver(t *testing.T, broker, topic string) *cekafka.Consumer {
	t.Helper()

	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Version = sarama.V2_6_0_0

	groupID := "test-consumer-group"
	receiver, err := cekafka.NewConsumer([]string{broker}, config, groupID, topic)
	require.NoError(t, err)

	// Open the consumer
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = receiver.OpenInbound(ctx)
	require.NoError(t, err)

	return receiver
}
