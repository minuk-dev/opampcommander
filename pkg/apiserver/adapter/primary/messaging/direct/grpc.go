package direct

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	commondirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/direct"
	servereventv1 "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/direct/gen/opampcommander/serverevent/v1"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
)

var _ Receiver = (*GRPCReceiver)(nil)

// GRPCReceiver serves incoming server events over gRPC.
type GRPCReceiver struct {
	address         string
	currentServerID string
	token           string
	logger          *slog.Logger
}

// NewGRPCReceiver creates a new GRPCReceiver bound to address. currentServerID rejects
// misrouted messages; a non-empty token requires senders to present a matching bearer.
func NewGRPCReceiver(address, currentServerID, token string, logger *slog.Logger) *GRPCReceiver {
	return &GRPCReceiver{
		address:         address,
		currentServerID: currentServerID,
		token:           token,
		logger:          logger,
	}
}

// Serve implements Receiver.
func (r *GRPCReceiver) Serve(ctx context.Context, handler agentport.ReceiveServerEventHandler) error {
	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", r.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", r.address, err)
	}

	server := grpc.NewServer()
	servereventv1.RegisterServerEventServiceServer(server, &grpcService{
		UnimplementedServerEventServiceServer: servereventv1.UnimplementedServerEventServiceServer{},
		currentServerID:                       r.currentServerID,
		token:                                 r.token,
		handler:                               handler,
		logger:                                r.logger,
	})

	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	r.logger.Info("starting direct gRPC receiver", slog.String("address", r.address))

	err = server.Serve(listener)
	if err != nil {
		return fmt.Errorf("direct gRPC receiver failed: %w", err)
	}

	return nil
}

// grpcService adapts the generated service interface to the domain handler.
type grpcService struct {
	servereventv1.UnimplementedServerEventServiceServer

	currentServerID string
	token           string
	handler         agentport.ReceiveServerEventHandler
	logger          *slog.Logger
}

// Deliver implements servereventv1.ServerEventServiceServer.
func (s *grpcService) Deliver(
	ctx context.Context,
	request *servereventv1.DeliverRequest,
) (*servereventv1.DeliverResponse, error) {
	if !s.authorized(ctx) {
		//nolint:wrapcheck // gRPC handlers return status errors verbatim so the code propagates.
		return nil, status.Error(codes.Unauthenticated, "invalid or missing credential")
	}

	messageType, err := commondirect.ParseMessageType(request.GetType())
	if err != nil {
		//nolint:wrapcheck // gRPC handlers return status errors verbatim so the code propagates.
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	target := request.GetTarget()
	if target != "" && target != s.currentServerID {
		s.logger.Warn("rejecting direct message addressed to another server",
			slog.String("target", target),
			slog.String("current", s.currentServerID))

		//nolint:wrapcheck // gRPC handlers return status errors verbatim so the code propagates.
		return nil, status.Error(codes.FailedPrecondition, "message addressed to another server")
	}

	payload, err := commondirect.DecodePayload(request.GetPayload())
	if err != nil {
		//nolint:wrapcheck // gRPC handlers return status errors verbatim so the code propagates.
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	message := serverevent.Message{
		Source:  request.GetSource(),
		Target:  target,
		Type:    messageType,
		Payload: payload,
	}

	err = s.handler(ctx, &message)
	if err != nil {
		s.logger.Warn("failed to handle direct message", slog.String("error", err.Error()))

		//nolint:wrapcheck // gRPC handlers return status errors verbatim so the code propagates.
		return nil, status.Error(codes.Internal, "failed to handle message")
	}

	return &servereventv1.DeliverResponse{}, nil
}

// authorized reports whether the RPC carries the required bearer credential. When no
// token is configured, all requests are accepted (trusted-network mode).
func (s *grpcService) authorized(ctx context.Context) bool {
	if s.token == "" {
		return true
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}

	values := md.Get(commondirect.AuthMetadataKey)
	if len(values) == 0 {
		return false
	}

	return commondirect.ConstantTimeTokenMatch(s.token, values[0])
}
