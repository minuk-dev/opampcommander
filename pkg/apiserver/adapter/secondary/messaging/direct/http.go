package direct

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	commondirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/direct"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
)

var _ Deliverer = (*HTTPDeliverer)(nil)

// errUnexpectedStatus is returned when a peer answers a delivery with a non-2xx status.
var errUnexpectedStatus = errors.New("peer returned unexpected status")

// defaultHTTPTimeout bounds a single direct delivery attempt.
const defaultHTTPTimeout = 5 * time.Second

// HTTPDeliverer delivers server events to peers over HTTP/JSON.
type HTTPDeliverer struct {
	client *http.Client
	logger *slog.Logger
}

// NewHTTPDeliverer creates a new HTTPDeliverer.
func NewHTTPDeliverer(logger *slog.Logger) *HTTPDeliverer {
	return &HTTPDeliverer{
		//exhaustruct:ignore
		client: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
		logger: logger,
	}
}

// Deliver implements Deliverer by POSTing the message envelope to the peer.
func (d *HTTPDeliverer) Deliver(ctx context.Context, address string, message serverevent.Message) error {
	envelope := commondirect.ToEnvelope(message)

	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to encode envelope: %w", err)
	}

	url := "http://" + address + commondirect.DeliverPath

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	// Drain the body so the connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: %d", errUnexpectedStatus, resp.StatusCode)
	}

	return nil
}

// Close releases idle connections.
func (d *HTTPDeliverer) Close() error {
	d.client.CloseIdleConnections()

	return nil
}
