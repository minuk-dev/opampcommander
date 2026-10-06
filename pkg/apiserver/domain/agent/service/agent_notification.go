package agentservice

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
)

var _ agentport.AgentNotificationUsecase = (*AgentNotificationService)(nil)

const (
	// DefaultNotificationFlushInterval is how often pending notifications are drained.
	DefaultNotificationFlushInterval = 100 * time.Millisecond
	// DefaultNotificationMaxBatchSize triggers an early flush for a single target once the
	// pending UID count for that target reaches this threshold. Acts as a soft cap.
	DefaultNotificationMaxBatchSize = 500
	// DefaultNotificationDispatchWorkers is the size of the worker pool that drains
	// coalesced batches in parallel so one slow target cannot starve the others. Two is
	// enough to keep a single slow/large target from blocking the rest at the production
	// scale this targets (~10 servers), while holding the per-node goroutine count down.
	DefaultNotificationDispatchWorkers = 2
	// DefaultNotificationDispatchQueue is the capacity of the dispatch channel feeding
	// the worker pool. Larger than the worker pool to absorb short bursts without
	// back-pressuring the flush loop.
	DefaultNotificationDispatchQueue = 64
	// DefaultNotificationShutdownCeiling is a hard upper bound on how long Run will
	// wait for in-flight workers to drain after ctx is cancelled. The primary
	// mechanism is parent ctx propagation; this ceiling only fires if a downstream
	// call (e.g. a Kafka producer flush) ignores ctx cancellation, in which case
	// we log a leak warning and let Run return so the FX executor isn't blocked.
	DefaultNotificationShutdownCeiling = 10 * time.Second

	unknownServerID = "unknown"
)

// AgentNotificationService handles notifications about agent updates.
//
// Notifications are coalesced per target server: when many agents on the same
// target server are updated in a short window, they are batched into a single
// inter-server message (one CloudEvent carrying multiple agent UIDs) instead
// of one message per agent. Identical UIDs within a batch are deduped.
//
// Dispatching of the coalesced batches runs in a small worker pool so that
// a single slow or large target cannot block the flush loop or other targets.
//
// NotifyAgentUpdated is asynchronous: it enqueues and returns nil immediately,
// so any downstream send error (target unreachable, kafka failure) is logged
// inside the dispatcher rather than surfaced to the original API caller.
type AgentNotificationService struct {
	serverMessageUsecase   agentport.ServerMessageUsecase
	serverUsecase          agentport.ServerUsecase
	serverIdentityProvider agentport.ServerIdentityProvider
	logger                 *slog.Logger

	flushInterval   time.Duration
	maxBatchSize    int
	shutdownCeiling time.Duration

	store   agentport.NotificationStore
	workers int
}

// NewAgentNotificationService creates a new instance of AgentNotificationService.
func NewAgentNotificationService(
	serverMessageUsecase agentport.ServerMessageUsecase,
	serverUsecase agentport.ServerUsecase,
	serverIdentityProvider agentport.ServerIdentityProvider,
	store agentport.NotificationStore,
	logger *slog.Logger,
) *AgentNotificationService {
	return &AgentNotificationService{
		serverMessageUsecase:   serverMessageUsecase,
		serverUsecase:          serverUsecase,
		serverIdentityProvider: serverIdentityProvider,
		logger:                 logger,
		flushInterval:          DefaultNotificationFlushInterval,
		maxBatchSize:           DefaultNotificationMaxBatchSize,
		shutdownCeiling:        DefaultNotificationShutdownCeiling,
		store:                  store,
		workers:                DefaultNotificationDispatchWorkers,
	}
}

// Name returns the name of the runner.
func (s *AgentNotificationService) Name() string {
	return "AgentNotificationService"
}

// Run drives the flush loop and a per-invocation dispatch worker pool. On
// cancellation workers stop within shutdownCeiling. Unprocessed notifications
// remain in the node-local Store; durable agent messages remain the recovery path.
func (s *AgentNotificationService) Run(ctx context.Context) error {
	var workers sync.WaitGroup
	for range s.workers {
		workers.Go(func() { ; s.dispatchWorker(ctx) })
	}

	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.waitForWorkers(&workers)

			return nil
		case <-ticker.C:
			s.flushAll(ctx)
		case <-s.store.FlushSignal():
			s.flushAll(ctx)
		}
	}
}

// NotifyAgentUpdated enqueues a pending-message notification for the agent's
// connected server. The actual inter-server message is sent later, after
// coalescing with other notifications targeting the same server.
//
// This call is fire-and-forget: enqueue is non-blocking and always returns nil.
// Downstream send failures (target server unreachable, Kafka error) surface as
// error-level logs inside the dispatch worker, NOT as a returned error — by the
// time we know about them, the original HTTP caller is long gone. Callers that
// need delivery guarantees must not rely on this method's return value.
func (s *AgentNotificationService) NotifyAgentUpdated(ctx context.Context, agent *agentmodel.Agent) error {
	logger := s.logger.With(
		slog.String("agentInstanceUID", agent.Metadata.InstanceUID.String()),
	)

	if !agent.HasPendingServerMessages() || !agent.IsConnected(ctx) {
		logger.Debug("no notification enqueued: no pending messages or agent not connected")

		return nil
	}

	serverID, err := agent.ConnectedServerID()
	if err != nil {
		logger.Warn("failed to get connected server ID", slog.String("error", err.Error()))

		return nil
	}

	if serverID == "" {
		logger.Debug("no notification enqueued: agent has no connected server ID")

		return nil
	}

	s.store.Enqueue(serverID, agent.Metadata.InstanceUID, s.maxBatchSize)

	return nil
}

// waitForWorkers waits for this Run invocation's workers to stop after cancellation.
func (s *AgentNotificationService) waitForWorkers(workers *sync.WaitGroup) {
	done := make(chan struct{})

	go func() {
		workers.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(s.shutdownCeiling):
		s.logger.Warn("dispatch workers did not drain within shutdown ceiling; abandoning",
			slog.Duration("ceiling", s.shutdownCeiling),
		)
	}
}

func (s *AgentNotificationService) flushAll(ctx context.Context) {
	sourceServerID := s.serverIdentityProvider.CurrentServerID()
	if sourceServerID == "" {
		sourceServerID = unknownServerID
	}

	for _, pending := range s.store.DrainPending() {
		err := s.store.EnqueueBatch(ctx, agentport.NotificationBatch{
			SourceServerID: sourceServerID, PendingNotification: pending,
		})
		if err != nil {
			s.logger.Warn("dropping batch during flush", slog.String("targetServerID", pending.TargetServerID),
				slog.Int("agentCount", len(pending.InstanceUIDs)), slog.String("error", err.Error()))

			return
		}
	}
}

func (s *AgentNotificationService) dispatchWorker(ctx context.Context) {
	for {
		batch, err := s.store.NextBatch(ctx)
		if err != nil {
			return
		}

		s.dispatchBatch(ctx, batch.SourceServerID, batch.TargetServerID, batch.InstanceUIDs)
	}
}

func (s *AgentNotificationService) dispatchBatch(
	ctx context.Context,
	sourceServerID string,
	targetServerID string,
	uids []uuid.UUID,
) {
	logger := s.logger.With(
		slog.String("targetServerID", targetServerID),
		slog.Int("agentCount", len(uids)),
	)

	// On shutdown the parent ctx is cancelled. Skip the downstream calls so they don't surface as flush
	// failures in the log; the batch's UIDs will be re-discovered on next start.
	if ctx.Err() != nil {
		logger.Debug("skipping dispatch: context done")

		return
	}

	server, err := s.serverUsecase.GetServer(ctx, targetServerID)
	if err != nil {
		logDispatchFailure(logger, "failed to dispatch notification: cannot get target server", err)

		return
	}

	err = s.serverMessageUsecase.SendMessageToServer(ctx, server, serverevent.Message{
		Source: sourceServerID,
		Target: targetServerID,
		Type:   serverevent.MessageTypeSendServerToAgent,
		Payload: serverevent.MessagePayload{
			MessageForServerToAgent: &serverevent.MessageForServerToAgent{
				TargetAgentInstanceUIDs: uids,
			},
			MessageForInvalidateAgentCache: nil,
		},
	})
	if err != nil {
		logDispatchFailure(logger, "failed to send batched notification", err)
	}
}

// logDispatchFailure emits at Error level normally, but downgrades to Debug when
// the underlying cause is ctx cancellation. The latter is expected on shutdown
// and should not page on-call; real downstream failures still surface loudly so
// operators correlating 'pending message stuck' alerts can find the cause.
func logDispatchFailure(logger *slog.Logger, msg string, err error) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		logger.Debug(msg+" (context done)", slog.String("error", err.Error()))

		return
	}

	logger.Error(msg, slog.String("error", err.Error()))
}
