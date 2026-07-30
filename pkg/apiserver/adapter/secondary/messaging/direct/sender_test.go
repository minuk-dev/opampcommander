package direct_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commondirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/direct"
	outdirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/messaging/direct"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
)

var errBoom = errors.New("boom")

// fakeDeliverer records the last delivery and returns configurable errors.
type fakeDeliverer struct {
	gotAddress string
	gotMessage serverevent.Message
	deliverErr error
	closeErr   error
	closed     bool
}

func (f *fakeDeliverer) Deliver(_ context.Context, address string, message serverevent.Message) error {
	f.gotAddress = address
	f.gotMessage = message

	return f.deliverErr
}

func (f *fakeDeliverer) Close() error {
	f.closed = true

	return f.closeErr
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func invalidateMessage() serverevent.Message {
	return serverevent.Message{
		Source: "server-a",
		Target: "server-b",
		Type:   serverevent.MessageTypeInvalidateAgentCache,
		Payload: serverevent.MessagePayload{
			MessageForServerToAgent: nil,
			MessageForInvalidateAgentCache: &serverevent.MessageForInvalidateAgentCache{
				AgentInstanceUIDs: []uuid.UUID{uuid.New()},
			},
		},
	}
}

func TestEventSenderAdapter_SendMessageToServer_Success(t *testing.T) {
	t.Parallel()

	deliverer := &fakeDeliverer{}
	sender := outdirect.NewEventSenderAdapter(deliverer, discardLogger())

	server := &agentmodel.Server{ID: "server-b", Address: "10.0.0.5:8081"}
	message := invalidateMessage()

	err := sender.SendMessageToServer(t.Context(), server, message)
	require.NoError(t, err)

	assert.Equal(t, "10.0.0.5:8081", deliverer.gotAddress)
	assert.Equal(t, message.Type, deliverer.gotMessage.Type)
}

func TestEventSenderAdapter_SendMessageToServer_DelivererError(t *testing.T) {
	t.Parallel()

	deliverer := &fakeDeliverer{deliverErr: errBoom}
	sender := outdirect.NewEventSenderAdapter(deliverer, discardLogger())
	server := &agentmodel.Server{ID: "server-b", Address: "10.0.0.5:8081"}

	err := sender.SendMessageToServer(t.Context(), server, invalidateMessage())
	require.ErrorIs(t, err, errBoom)
}

func TestEventSenderAdapter_SendMessageToServer_NoAddress(t *testing.T) {
	t.Parallel()

	deliverer := &fakeDeliverer{}
	sender := outdirect.NewEventSenderAdapter(deliverer, discardLogger())
	server := &agentmodel.Server{ID: "server-b", Address: ""}

	err := sender.SendMessageToServer(t.Context(), server, invalidateMessage())
	require.ErrorIs(t, err, outdirect.ErrNoPeerAddress)
	assert.Empty(t, deliverer.gotAddress, "deliverer must not be called without an address")
}

func TestEventSenderAdapter_Close(t *testing.T) {
	t.Parallel()

	t.Run("delegates", func(t *testing.T) {
		t.Parallel()

		deliverer := &fakeDeliverer{}
		sender := outdirect.NewEventSenderAdapter(deliverer, discardLogger())

		require.NoError(t, sender.Close())
		assert.True(t, deliverer.closed)
	})

	t.Run("propagates error", func(t *testing.T) {
		t.Parallel()

		deliverer := &fakeDeliverer{closeErr: errBoom}
		sender := outdirect.NewEventSenderAdapter(deliverer, discardLogger())

		require.ErrorIs(t, sender.Close(), errBoom)
	})
}

func TestHTTPDeliverer_Deliver_Success(t *testing.T) {
	t.Parallel()

	var (
		gotAuth string
		gotPath string
		gotBody commondirect.Envelope
	)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuth = request.Header.Get(commondirect.AuthHeader)
		gotPath = request.URL.Path
		_ = json.NewDecoder(request.Body).Decode(&gotBody)

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	deliverer := outdirect.NewHTTPDeliverer("s3cret", discardLogger())

	defer func() { _ = deliverer.Close() }()

	message := invalidateMessage()
	err := deliverer.Deliver(t.Context(), strings.TrimPrefix(server.URL, "http://"), message)
	require.NoError(t, err)

	assert.Equal(t, commondirect.BearerPrefix+"s3cret", gotAuth)
	assert.Equal(t, commondirect.DeliverPath, gotPath)
	assert.Equal(t, message.Source, gotBody.Source)
}

func TestHTTPDeliverer_Deliver_NoTokenOmitsHeader(t *testing.T) {
	t.Parallel()

	gotAuth := "unset"

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuth = request.Header.Get(commondirect.AuthHeader)

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	deliverer := outdirect.NewHTTPDeliverer("", discardLogger())

	defer func() { _ = deliverer.Close() }()

	err := deliverer.Deliver(t.Context(), strings.TrimPrefix(server.URL, "http://"), invalidateMessage())
	require.NoError(t, err)
	assert.Empty(t, gotAuth)
}

func TestHTTPDeliverer_Deliver_ErrorStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	deliverer := outdirect.NewHTTPDeliverer("", discardLogger())

	defer func() { _ = deliverer.Close() }()

	err := deliverer.Deliver(t.Context(), strings.TrimPrefix(server.URL, "http://"), invalidateMessage())
	require.Error(t, err)
}

func TestHTTPDeliverer_Deliver_Unreachable(t *testing.T) {
	t.Parallel()

	deliverer := outdirect.NewHTTPDeliverer("", discardLogger())

	defer func() { _ = deliverer.Close() }()

	// 127.0.0.1:1 is reserved and refuses connections.
	err := deliverer.Deliver(t.Context(), "127.0.0.1:1", invalidateMessage())
	require.Error(t, err)
}
