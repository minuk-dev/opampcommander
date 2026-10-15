package kafka

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

func TestSender_Observability(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		producer := &testAsyncProducer{
			input: make(chan *sarama.ProducerMessage, 1), failures: make(chan *sarama.ProducerError),
			successes: make(chan *sarama.ProducerMessage),
		}

		sender := NewSender(producer, "events", slog.New(slog.DiscardHandler), time.Second)
		defer func() { require.NoError(t, sender.Close(t.Context())) }()

		reader := sdkmetric.NewManualReader()

		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		defer func() { require.NoError(t, provider.Shutdown(t.Context())) }()

		require.NoError(t, sender.Observe(provider.Meter("test")))
		adapter, err := NewEventSenderAdapter(sender)
		require.NoError(t, err)

		message := serverevent.Message{Target: "remote", Type: serverevent.MessageTypeInvalidateAgentCache}
		require.NoError(t, adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "remote"}, message))
		assertDeliveryMetrics(t, reader, map[string]int64{"enqueued": 1, "delivered": 0, "failed": 0, "rejected": 0}, 0)

		producer.failures <- &sarama.ProducerError{Msg: <-producer.input, Err: sarama.ErrOutOfBrokers}

		synctest.Wait()
		assertDeliveryMetrics(t, reader, map[string]int64{"enqueued": 1, "delivered": 0, "failed": 1, "rejected": 0}, 1)
		require.True(t, sender.Health(t.Context()).Degraded)

		helper := healthcheck.NewHealthHelper([]healthcheck.HealthIndicator{sender})
		response := httptest.NewRecorder()
		helper.IsHealth(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil))
		require.Equal(t, http.StatusOK, response.Code)
		require.Contains(t, response.Body.String(), "KafkaProducer")
		require.Contains(t, response.Body.String(), sarama.ErrOutOfBrokers.Error())
		response = httptest.NewRecorder()
		helper.IsReady(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))
		require.Equal(t, http.StatusOK, response.Code)

		// Queue acceptance alone must not clear a terminal delivery failure.
		require.NoError(t, adapter.SendMessageToServer(t.Context(), &agentmodel.Server{ID: "remote"}, message))
		require.True(t, sender.Health(t.Context()).Degraded)

		producer.successes <- <-producer.input

		synctest.Wait()
		require.False(t, sender.Health(t.Context()).Degraded)
		require.Empty(t, sender.Health(t.Context()).Reason)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, adapter.SendMessageToServer(ctx, &agentmodel.Server{ID: "remote"}, message), context.Canceled)
		assertDeliveryMetrics(t, reader, map[string]int64{"enqueued": 2, "delivered": 1, "failed": 1, "rejected": 1}, 0)
	})
}

func assertDeliveryMetrics(t *testing.T, reader *sdkmetric.ManualReader, outcomes map[string]int64, degraded int64) {
	t.Helper()

	var data metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &data))
	require.Len(t, data.ScopeMetrics, 1)
	require.Len(t, data.ScopeMetrics[0].Metrics, 2)

	for _, measurement := range data.ScopeMetrics[0].Metrics {
		switch measurement.Name {
		case "opampcommander.kafka.events":
			values, ok := measurement.Data.(metricdata.Sum[int64])
			require.True(t, ok)
			require.True(t, values.IsMonotonic)

			actual := make(map[string]int64)

			for _, point := range values.DataPoints {
				outcome, exists := point.Attributes.Value(attribute.Key("outcome"))
				require.True(t, exists)

				actual[outcome.AsString()] = point.Value
			}

			require.Equal(t, outcomes, actual)
		case "opampcommander.kafka.delivery_degraded":
			values, ok := measurement.Data.(metricdata.Gauge[int64])
			require.True(t, ok)
			require.Len(t, values.DataPoints, 1)
			require.Equal(t, degraded, values.DataPoints[0].Value)
		default:
			t.Fatalf("unexpected metric %s", measurement.Name)
		}
	}
}
