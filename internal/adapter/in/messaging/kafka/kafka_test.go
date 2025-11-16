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

	defer func() {
		terminateCtx, terminateCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer terminateCancel()

		_ = kafkaContainer.Terminate(terminateCtx)
	}()

	// Given: Kafka sender and receiver are configured
	topic := "test.opampcommander.events"
	sender := createTestSender(t, broker, topic)
	receiver := createTestReceiver(t, broker, topic)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	adapter, err := kafka.NewEventSenderAdapter(sender, receiver, logger)
	require.NoError(t, err)

	// Given: Receiver is started
	receivedMessages := make(chan *serverevent.Message, 10)

	receiverCtx, receiverCancel := context.WithCancel(ctx)
	defer receiverCancel()

	go func() {
		_ = adapter.StartReceiver(receiverCtx, func(_ context.Context, msg *serverevent.Message) error {
			receivedMessages <- msg

			return nil
		})
	}()

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

	// When: Message is sent
	err = adapter.SendMessageToServer(ctx, "test-server-1", testMessage)
	require.NoError(t, err)

	// Then: Message should be received
	assert.Eventually(t, func() bool {
		select {
		case received := <-receivedMessages:
			assert.Equal(t, testMessage.Type, received.Type)
			require.NotNil(t, received.Payload.MessageForServerToAgent)
			assert.Equal(t, testAgentUID, received.Payload.TargetAgentInstanceUIDs[0])
			assert.Contains(t, received.Source, "test-server-1")

			return true
		default:
			return false
		}
	}, 30*time.Second, 100*time.Millisecond, "Message should be received")
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

	defer func() {
		terminateCtx, terminateCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer terminateCancel()

		_ = kafkaContainer.Terminate(terminateCtx)
	}()

	topic := "test.opampcommander.multi"
	sender := createTestSender(t, broker, topic)
	receiver := createTestReceiver(t, broker, topic)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	adapter, err := kafka.NewEventSenderAdapter(sender, receiver, logger)
	require.NoError(t, err)

	receivedMessages := make(chan *serverevent.Message, 10)

	receiverCtx, receiverCancel := context.WithCancel(ctx)
	defer receiverCancel()

	go func() {
		_ = adapter.StartReceiver(receiverCtx, func(_ context.Context, msg *serverevent.Message) error {
			receivedMessages <- msg

			return nil
		})
	}()

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
	assert.Eventually(t, func() bool {
		receivedCount := 0

		for {
			select {
			case msg := <-receivedMessages:
				if msg != nil {
					receivedCount++
				}

				if receivedCount == numMessages {
					return true
				}
			default:
				return receivedCount == numMessages
			}
		}
	}, 30*time.Second, 100*time.Millisecond, "All messages should be received")
}

// Helper functions

//nolint:ireturn
func startKafkaContainer(t *testing.T) (testcontainers.Container, string) {
	t.Helper()

	kafkaContainer, err := kafkaTestContainer.Run(t.Context(),
		"confluentinc/cp-kafka:7.5.0",
		kafkaTestContainer.WithClusterID("test-cluster-id"),
	)
	require.NoError(t, err)

	brokers, err := kafkaContainer.Brokers(t.Context())
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

	groupID := "test-consumer-group-" + uuid.New().String()
	receiver, err := cekafka.NewConsumer([]string{broker}, config, groupID, topic)
	require.NoError(t, err)

	return receiver
}
