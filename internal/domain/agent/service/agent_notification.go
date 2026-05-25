package agentservice

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/agent/model/serverevent"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
)

var _ agentport.AgentNotificationUsecase = (*AgentNotificationService)(nil)

const (
	// DefaultNotificationFlushInterval is how often pending notifications are drained.
	DefaultNotificationFlushInterval = 100 * time.Millisecond
	// DefaultNotificationMaxBatchSize triggers an early flush for a single target once the
	// pending UID count for that target reaches this threshold. Acts as a soft cap.
	DefaultNotificationMaxBatchSize = 500
	// DefaultNotificationDispatchWorkers is the size of the worker pool that drains
	// coalesced batches in parallel so one slow target cannot starve the others.
	DefaultNotificationDispatchWorkers = 4
	// DefaultNotificationDispatchQueue is the capacity of the dispatch channel feeding
	// the worker pool. Larger than the worker pool to absorb short bursts without
	// back-pressuring the flush loop.
	DefaultNotificationDispatchQueue = 64
	// DefaultNotificationShutdownTimeout caps the final flush on graceful shutdown so
	// an unreachable backend cannot block the executor's WaitGroup indefinitely.
	DefaultNotificationShutdownTimeout = 5 * time.Second
	// notificationFlushSignalBuffer is the buffer size of the early-flush signal channel.
	notificationFlushSignalBuffer = 1

	unknownServerID = "unknown"
)

type notificationBatch struct {
	sourceServerID string
	targetServerID string
	uids           []uuid.UUID
}

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
	shutdownTimeout time.Duration

	mu       sync.Mutex
	pending  map[string]map[uuid.UUID]struct{}
	flushNow chan struct{}

	dispatchCh chan notificationBatch
	workerWG   sync.WaitGroup
	workers    int
}

// NewAgentNotificationService creates a new instance of AgentNotificationService.
func NewAgentNotificationService(
	serverMessageUsecase agentport.ServerMessageUsecase,
	serverUsecase agentport.ServerUsecase,
	serverIdentityProvider agentport.ServerIdentityProvider,
	logger *slog.Logger,
) *AgentNotificationService {
	return &AgentNotificationService{
		serverMessageUsecase:   serverMessageUsecase,
		serverUsecase:          serverUsecase,
		serverIdentityProvider: serverIdentityProvider,
		logger:                 logger,
		flushInterval:          DefaultNotificationFlushInterval,
		maxBatchSize:           DefaultNotificationMaxBatchSize,
		shutdownTimeout:        DefaultNotificationShutdownTimeout,
		pending:                make(map[string]map[uuid.UUID]struct{}),
		flushNow:               make(chan struct{}, notificationFlushSignalBuffer),
		dispatchCh:             make(chan notificationBatch, DefaultNotificationDispatchQueue),
		workers:                DefaultNotificationDispatchWorkers,
	}
}

// Name returns the name of the runner.
func (s *AgentNotificationService) Name() string {
	return "AgentNotificationService"
}

// Run drives the periodic flush loop and the dispatch worker pool. It exits
// cleanly (nil) when ctx is cancelled, performing a best-effort final flush
// bounded by shutdownTimeout and then draining any in-flight batches.
func (s *AgentNotificationService) Run(ctx context.Context) error {
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()

	for range s.workers {
		s.workerWG.Add(1)

		go s.dispatchWorker(workerCtx)
	}

	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.gracefulShutdown(cancelWorkers)

			return nil
		case <-ticker.C:
			s.flushAll(ctx)
		case <-s.flushNow:
			s.flushAll(ctx)
		}
	}
}

// gracefulShutdown performs a bounded final flush, then signals workers to exit
// and waits for them. Bounding the final flush ensures an unreachable backend
// (MongoDB, Kafka) cannot stall the executor's WaitGroup past the operator's
// FX OnStop deadline.
func (s *AgentNotificationService) gracefulShutdown(cancelWorkers context.CancelFunc) {
	flushCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	s.flushAll(flushCtx)

	close(s.dispatchCh)
	// Give workers a small extra grace window after the channel closes; if any worker
	// is mid-dispatch on a stuck backend, cancel its context too.
	done := make(chan struct{})
	go func() {
		s.workerWG.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-flushCtx.Done():
		cancelWorkers()
		<-done
	}
}

// NotifyAgentUpdated enqueues a pending-message notification for the agent's
// connected server. The actual inter-server message is sent later, after
// coalescing with other notifications targeting the same server.
func (s *AgentNotificationService) NotifyAgentUpdated(_ context.Context, agent *agentmodel.Agent) error {
	logger := s.logger.With(
		slog.String("agentInstanceUID", agent.Metadata.InstanceUID.String()),
	)

	if !agent.HasPendingServerMessages() || !agent.IsConnected(context.Background()) {
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

	s.enqueue(serverID, agent.Metadata.InstanceUID)

	return nil
}

// RestartAgent requests the agent to restart.
func (s *AgentNotificationService) RestartAgent(_ context.Context, instanceUID uuid.UUID) error {
	s.logger.Info("restart notification triggered", "instanceUID", instanceUID.String())
	// This method is now primarily for logging/monitoring purposes
	// The actual restart logic is handled in the application service layer
	return nil
}

func (s *AgentNotificationService) enqueue(serverID string, instanceUID uuid.UUID) {
	s.mu.Lock()
	bucket, ok := s.pending[serverID]
	if !ok {
		bucket = make(map[uuid.UUID]struct{})
		s.pending[serverID] = bucket
	}

	bucket[instanceUID] = struct{}{}
	size := len(bucket)
	s.mu.Unlock()

	if size >= s.maxBatchSize {
		// Non-blocking nudge: at most one pending flush signal at a time.
		select {
		case s.flushNow <- struct{}{}:
		default:
		}
	}
}

func (s *AgentNotificationService) flushAll(ctx context.Context) {
	s.mu.Lock()
	if len(s.pending) == 0 {
		s.mu.Unlock()

		return
	}

	pending := s.pending
	s.pending = make(map[string]map[uuid.UUID]struct{})
	s.mu.Unlock()

	sourceServerID := s.serverIdentityProvider.CurrentServerID()
	if sourceServerID == "" {
		sourceServerID = unknownServerID
	}

	for targetServerID, uidSet := range pending {
		if len(uidSet) == 0 {
			continue
		}

		uids := make([]uuid.UUID, 0, len(uidSet))
		for u := range uidSet {
			uids = append(uids, u)
		}

		batch := notificationBatch{
			sourceServerID: sourceServerID,
			targetServerID: targetServerID,
			uids:           uids,
		}

		select {
		case s.dispatchCh <- batch:
		case <-ctx.Done():
			s.logger.Warn("dropping batch during flush: context done",
				slog.String("targetServerID", targetServerID),
				slog.Int("agentCount", len(uids)),
			)

			return
		}
	}
}

func (s *AgentNotificationService) dispatchWorker(ctx context.Context) {
	defer s.workerWG.Done()

	for batch := range s.dispatchCh {
		s.dispatchBatch(ctx, batch.sourceServerID, batch.targetServerID, batch.uids)
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

	server, err := s.serverUsecase.GetServer(ctx, targetServerID)
	if err != nil {
		logger.Warn("failed to flush notifications: cannot get target server",
			slog.String("error", err.Error()),
		)

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
		},
	})
	if err != nil {
		logger.Warn("failed to send batched notification",
			slog.String("error", err.Error()),
		)
	}
}
