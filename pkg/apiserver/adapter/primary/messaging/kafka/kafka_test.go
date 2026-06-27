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

	kafkamodel "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/kafka"
	inkafka "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/messaging/kafka"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestEventReceiverAdapter_StartReceiver(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()

	// Given: Kafka is running
	kafkaContainer, broker := startKafkaContainer(ctx, t)

	defer func() { _ = kafkaContainer.Terminate(ctx) }()

	topic := "test-topic-" + uuid.New().String()

	// Given: Mock server identity provider
	currentServerID := "test-server-id"
	identityProvider := &mockServerIdentityProvider{
		serverID: currentServerID,
	}

	// Given: EventReceiverAdapter is created
	consumer := createTestConsumer(t, broker, topic)
	logger := slog.New(slog.NewTextHandler(testutil.TestLogWriter{T: t}, nil))
	adapter, err := inkafka.NewEventReceiverAdapter(identityProvider, consumer, logger)
	require.NoError(t, err)

	// Given: Handler to capture received messages
	receivedMessages := make(chan *serverevent.Message, 10)
	handlerErr := make(chan error, 1)
	handler := func(_ context.Context, msg *serverevent.Message) error {
		select {
		case receivedMessages <- msg:
			return nil
		case err := <-handlerErr:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Given: Start receiver in background
	receiverCtx, cancelReceiver := context.WithCancel(ctx)
	defer cancelReceiver()

	go func() {
		err := adapter.StartReceiver(receiverCtx, handler)
		if err != nil && receiverCtx.Err() == nil {
			t.Logf("Receiver stopped with error: %v", err)
		}
	}()

	// Wait for receiver to be ready
	time.Sleep(2 * time.Second)

	// When: Send message via Kafka producer
	sender := createTestSender(t, broker, topic)
	agentUID := uuid.New()
	sendTestMessage(ctx, t, sender, currentServerID, agentUID)

	// Then: Message should be received
	select {
	case received := <-receivedMessages:
		assert.Equal(t, currentServerID, received.Target)
		assert.Equal(t, serverevent.MessageTypeSendServerToAgent, received.Type)
		assert.NotNil(t, received.Payload.MessageForServerToAgent)
		assert.Len(t, received.Payload.TargetAgentInstanceUIDs, 1)
		assert.Equal(t, agentUID, received.Payload.TargetAgentInstanceUIDs[0])
	case <-time.After(15 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestEventReceiverAdapter_FiltersByTargetServer(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()

	// Given: Kafka is running
	kafkaContainer, broker := startKafkaContainer(ctx, t)

	defer func() { _ = kafkaContainer.Terminate(ctx) }()

	topic := "test-topic-" + uuid.New().String()

	// Given: Current server ID
	currentServerID := "server-1"
	otherServerID := "server-2"
	identityProvider := &mockServerIdentityProvider{
		serverID: currentServerID,
	}

	// Given: EventReceiverAdapter is created
	consumer := createTestConsumer(t, broker, topic)
	logger := slog.New(slog.NewTextHandler(testutil.TestLogWriter{T: t}, nil))
	adapter, err := inkafka.NewEventReceiverAdapter(identityProvider, consumer, logger)
	require.NoError(t, err)

	// Given: Handler to capture received messages
	receivedMessages := make(chan *serverevent.Message, 10)
	handlerErr := make(chan error, 1)
	handler := func(_ context.Context, msg *serverevent.Message) error {
		select {
		case receivedMessages <- msg:
			return nil
		case err := <-handlerErr:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Given: Start receiver in background
	receiverCtx, cancelReceiver := context.WithCancel(ctx)
	defer cancelReceiver()

	go func() {
		err := adapter.StartReceiver(receiverCtx, handler)
		if err != nil && receiverCtx.Err() == nil {
			t.Logf("Receiver stopped with error: %v", err)
		}
	}()

	// Wait for receiver to be ready
	time.Sleep(2 * time.Second)

	// When: Send messages to both servers
	sender := createTestSender(t, broker, topic)

	// Two messages for the current server and one for another server (which must be filtered).
	currentAgentUID1 := uuid.New()
	currentAgentUID2 := uuid.New()
	otherAgentUID := uuid.New()

	sendTestMessage(ctx, t, sender, currentServerID, currentAgentUID1)
	sendTestMessage(ctx, t, sender, otherServerID, otherAgentUID)
	sendTestMessage(ctx, t, sender, currentServerID, currentAgentUID2)

	// Then: both current-server messages should be received. Kafka only guarantees ordering
	// within a partition, so assert on the set of received agent UIDs rather than their order.
	receivedUIDs := make(map[uuid.UUID]struct{}, 2)
	timeout := time.After(15 * time.Second)

	for len(receivedUIDs) < 2 {
		select {
		case received := <-receivedMessages:
			assert.Equal(t, currentServerID, received.Target, "Should only receive messages for current server")
			require.Len(t, received.Payload.TargetAgentInstanceUIDs, 1)
			receivedUIDs[received.Payload.TargetAgentInstanceUIDs[0]] = struct{}{}
		case <-timeout:
			t.Fatalf("Timeout: received %d/2 messages", len(receivedUIDs))
		}
	}

	assert.Contains(t, receivedUIDs, currentAgentUID1, "first current-server message should be received")
	assert.Contains(t, receivedUIDs, currentAgentUID2, "second current-server message should be received")
	assert.NotContains(t, receivedUIDs, otherAgentUID, "other-server message must be filtered out")

	// Ensure no more messages are received (the one for other server should be filtered)
	select {
	case msg := <-receivedMessages:
		t.Fatalf("Should not receive message for other server: %+v", msg)
	case <-time.After(2 * time.Second):
		// Expected: no more messages
	}
}

// Helper functions

type mockServerIdentityProvider struct {
	serverID string
}

func (m *mockServerIdentityProvider) CurrentServer(_ context.Context) (*agentmodel.Server, error) {
	return &agentmodel.Server{
		ID: m.serverID,
	}, nil
}

func (m *mockServerIdentityProvider) CurrentServerID() string {
	return m.serverID
}

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

func sendTestMessage(
	ctx context.Context,
	t *testing.T,
	sender *cekafka.Sender,
	targetServerID string,
	agentUID uuid.UUID,
) {
	t.Helper()

	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetSource("opampcommander/server/test-source")
	event.SetSubject(targetServerID)
	event.SetType(kafkamodel.SendToAgentEventType)
	event.SetSpecVersion(kafkamodel.CloudEventMessageSpec)

	payload := serverevent.MessagePayload{
		MessageForServerToAgent: &serverevent.MessageForServerToAgent{
			TargetAgentInstanceUIDs: []uuid.UUID{agentUID},
		},
	}

	err := event.SetData(kafkamodel.CloudEventContentType, payload)
	require.NoError(t, err)

	ceClient, err := cloudevents.NewClient(sender)
	require.NoError(t, err)

	err = ceClient.Send(ctx, event)
	require.NoError(t, err)
}

var _ agentport.ServerIdentityProvider = (*mockServerIdentityProvider)(nil)
