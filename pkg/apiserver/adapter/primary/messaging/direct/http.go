package direct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	commondirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/direct"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ Receiver = (*HTTPReceiver)(nil)

// defaultReadHeaderTimeout bounds how long a client may take to send request headers.
const defaultReadHeaderTimeout = 5 * time.Second

// defaultShutdownTimeout bounds graceful shutdown of the receiver server.
const defaultShutdownTimeout = 5 * time.Second

// maxBodyBytes caps an incoming delivery body. Payloads are small lists of UUIDs.
const maxBodyBytes = 1 << 20 // 1 MiB

// HTTPReceiver serves incoming server events over HTTP/JSON.
type HTTPReceiver struct {
	address         string
	currentServerID string
	token           string
	logger          *slog.Logger
}

// NewHTTPReceiver creates a new HTTPReceiver bound to address. currentServerID rejects
// misrouted messages; a non-empty token requires senders to present a matching bearer.
func NewHTTPReceiver(address, currentServerID, token string, logger *slog.Logger) *HTTPReceiver {
	return &HTTPReceiver{
		address:         address,
		currentServerID: currentServerID,
		token:           token,
		logger:          logger,
	}
}

// Serve implements Receiver.
func (r *HTTPReceiver) Serve(ctx context.Context, handler agentport.ReceiveServerEventHandler) error {
	mux := http.NewServeMux()
	mux.HandleFunc(commondirect.DeliverPath, r.handleDeliver(handler))

	//exhaustruct:ignore
	server := &http.Server{
		Addr:              r.address,
		Handler:           mux,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	go func() {
		<-ctx.Done()

		// ctx is already cancelled here; keep its values but drop cancellation so the
		// graceful shutdown gets its own timeout window.
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultShutdownTimeout)
		defer cancel()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			r.logger.Warn("failed to gracefully shut down direct HTTP receiver",
				slog.String("error", err.Error()))
		}
	}()

	r.logger.Info("starting direct HTTP receiver", slog.String("address", r.address))

	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("direct HTTP receiver failed: %w", err)
	}

	return nil
}

func (r *HTTPReceiver) handleDeliver(
	handler agentport.ReceiveServerEventHandler,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)

			return
		}

		if !r.authorized(request) {
			http.Error(writer, "unauthorized", http.StatusUnauthorized)

			return
		}

		request.Body = http.MaxBytesReader(writer, request.Body, maxBodyBytes)

		var envelope commondirect.Envelope

		err := json.NewDecoder(request.Body).Decode(&envelope)
		if err != nil {
			http.Error(writer, "invalid body", http.StatusBadRequest)

			return
		}

		message, err := envelope.ToMessage()
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)

			return
		}

		if message.Target != "" && message.Target != r.currentServerID {
			r.logger.Warn("rejecting direct message addressed to another server",
				slog.String("target", message.Target),
				slog.String("current", r.currentServerID))
			http.Error(writer, "message addressed to another server", http.StatusConflict)

			return
		}

		err = handler(request.Context(), &message)
		if err != nil {
			r.logger.Warn("failed to handle direct message", slog.String("error", err.Error()))
			http.Error(writer, "failed to handle message", http.StatusInternalServerError)

			return
		}

		writer.WriteHeader(http.StatusNoContent)
	}
}

// authorized reports whether the request carries the required bearer credential. When no
// token is configured, all requests are accepted (trusted-network mode).
func (r *HTTPReceiver) authorized(request *http.Request) bool {
	if r.token == "" {
		return true
	}

	return commondirect.ConstantTimeTokenMatch(r.token, request.Header.Get(commondirect.AuthHeader))
}
