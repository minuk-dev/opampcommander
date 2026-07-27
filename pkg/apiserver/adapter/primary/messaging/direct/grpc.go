package direct

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commondirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/direct"
	servereventv1 "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/direct/gen/opampcommander/serverevent/v1"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
)

var _ Receiver = (*GRPCReceiver)(nil)

// GRPCReceiver serves incoming server events over gRPC.
type GRPCReceiver struct {
	address string
	logger  *slog.Logger
}

// NewGRPCReceiver creates a new GRPCReceiver bound to address.
func NewGRPCReceiver(address string, logger *slog.Logger) *GRPCReceiver {
	return &GRPCReceiver{
		address: address,
		logger:  logger,
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

	handler agentport.ReceiveServerEventHandler
	logger  *slog.Logger
}

// Deliver implements servereventv1.ServerEventServiceServer.
func (s *grpcService) Deliver(
	ctx context.Context,
	request *servereventv1.DeliverRequest,
) (*servereventv1.DeliverResponse, error) {
	messageType, err := commondirect.ParseMessageType(request.GetType())
	if err != nil {
		//nolint:wrapcheck // gRPC handlers return status errors verbatim so the code propagates.
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	payload, err := commondirect.DecodePayload(request.GetPayload())
	if err != nil {
		//nolint:wrapcheck // gRPC handlers return status errors verbatim so the code propagates.
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	message := serverevent.Message{
		Source:  request.GetSource(),
		Target:  request.GetTarget(),
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
