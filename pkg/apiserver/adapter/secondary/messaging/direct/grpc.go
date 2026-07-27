package direct

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	commondirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/direct"
	servereventv1 "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/direct/gen/opampcommander/serverevent/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
)

var _ Deliverer = (*GRPCDeliverer)(nil)

// GRPCDeliverer delivers server events to peers over gRPC. Client connections are
// created lazily per peer address and reused across deliveries.
//
// Transport security is currently insecure (plaintext), consistent with in-cluster
// pod-to-pod traffic; mTLS between peers is left as a follow-up.
type GRPCDeliverer struct {
	logger *slog.Logger

	mu    sync.Mutex
	conns map[string]*grpc.ClientConn
}

// NewGRPCDeliverer creates a new GRPCDeliverer.
func NewGRPCDeliverer(logger *slog.Logger) *GRPCDeliverer {
	return &GRPCDeliverer{
		logger: logger,
		mu:     sync.Mutex{},
		conns:  make(map[string]*grpc.ClientConn),
	}
}

// Deliver implements Deliverer by invoking the peer's ServerEventService.Deliver RPC.
func (d *GRPCDeliverer) Deliver(ctx context.Context, address string, message serverevent.Message) error {
	conn, err := d.connFor(address)
	if err != nil {
		return fmt.Errorf("failed to connect to peer %s: %w", address, err)
	}

	payload, err := commondirect.EncodePayload(message.Payload)
	if err != nil {
		return fmt.Errorf("failed to encode payload: %w", err)
	}

	client := servereventv1.NewServerEventServiceClient(conn)

	_, err = client.Deliver(ctx, &servereventv1.DeliverRequest{
		Source:  message.Source,
		Target:  message.Target,
		Type:    message.Type.String(),
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("failed to deliver via gRPC to %s: %w", address, err)
	}

	return nil
}

// Close closes all cached client connections.
func (d *GRPCDeliverer) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for address, conn := range d.conns {
		err := conn.Close()
		if err != nil {
			d.logger.Warn("failed to close gRPC connection",
				slog.String("address", address),
				slog.String("error", err.Error()))
		}
	}

	d.conns = make(map[string]*grpc.ClientConn)

	return nil
}

// connFor returns a cached client connection for the address, dialing lazily on first use.
func (d *GRPCDeliverer) connFor(address string) (*grpc.ClientConn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if conn, ok := d.conns[address]; ok {
		return conn, nil
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	d.conns[address] = conn

	return conn, nil
}
