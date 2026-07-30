package direct_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	commondirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/direct"
	indirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/messaging/direct"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
)

var errHandlerFailed = errors.New("handler failed")

// startHTTPReceiver starts an HTTP direct receiver with the given handler and returns its
// base URL once it is accepting connections.
func startHTTPReceiver(
	ctx context.Context,
	t *testing.T,
	handler agentport.ReceiveServerEventHandler,
) string {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)
	//nolint:contextcheck // freeAddress reserves an ephemeral port; there is no serve ctx to thread.
	address := freeAddress(t)
	receiver := indirect.NewEventReceiverAdapter(
		indirect.NewHTTPReceiver(address, targetServerID, "", logger), logger)

	go func() { _ = receiver.StartReceiver(ctx, handler) }()

	requireServing(ctx, t, address)

	return "http://" + address + commondirect.DeliverPath
}

func post(ctx context.Context, t *testing.T, url string, body []byte) int {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode
}

func TestHTTPReceiver_RejectsWrongMethod(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	url := startHTTPReceiver(ctx, t, capturingHandler(make(chan *serverevent.Message, 1)))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestHTTPReceiver_RejectsInvalidBody(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	url := startHTTPReceiver(ctx, t, capturingHandler(make(chan *serverevent.Message, 1)))

	require.Equal(t, http.StatusBadRequest, post(ctx, t, url, []byte("{not json")))
}

func TestHTTPReceiver_HandlerErrorReturns500(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	failing := func(_ context.Context, _ *serverevent.Message) error {
		return errHandlerFailed
	}
	url := startHTTPReceiver(ctx, t, failing)

	body, err := json.Marshal(commondirect.ToEnvelope(sampleMessage()))
	require.NoError(t, err)

	require.Equal(t, http.StatusInternalServerError, post(ctx, t, url, body))
}
