package agentservice

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model/serverevent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/datastructure/sets"
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
	// DefaultNotificationShutdownCeiling is a hard upper bound on how long Run will
	// wait for in-flight workers to drain after ctx is cancelled. The primary
	// mechanism is parent ctx propagation; this ceiling only fires if a downstream
	// call (e.g. a Kafka producer flush) ignores ctx cancellation, in which case
	// we log a leak warning and let Run return so the FX executor isn't blocked.
	DefaultNotificationShutdownCeiling = 10 * time.Second
	// notificationFlushSignalBuffer is the buffer size of the early-flush signal channel.
	notificationFlushSignalBuffer = 1

	unknownServerID = "unknown"
)

type notificationBatch struct {
	sourceServerID string
	targetServerID string
	uids           []uuid.UUID
}

// pendingBucket holds the deduped UIDs pending dispatch for a single target server.
// Each bucket has its own lock so writes against different target servers don't
// contend on a single shared mutex.
type pendingBucket struct {
	mu   sync.Mutex
	uids sets.UUID
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
	shutdownCeiling time.Duration

	// pending maps target serverID (string) -> *pendingBucket.
	// sync.Map avoids a single outer lock when many goroutines enqueue
	// notifications for different target servers concurrently. Each bucket
	// carries its own mutex protecting its UUID set.
	pending  sync.Map
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
		shutdownCeiling:        DefaultNotificationShutdownCeiling,
		pending:                sync.Map{},
		flushNow:               make(chan struct{}, notificationFlushSignalBuffer),
		dispatchCh:             make(chan notificationBatch, DefaultNotificationDispatchQueue),
		workerWG:               sync.WaitGroup{},
		workers:                DefaultNotificationDispatchWorkers,
	}
}

// Name returns the name of the runner.
func (s *AgentNotificationService) Name() string {
	return "AgentNotificationService"
}

// Run drives the periodic flush loop and the dispatch worker pool. It exits
// cleanly (nil) when ctx is cancelled, then closes the dispatch channel and
// waits for the workers to drain. Any in-flight pending notifications that
// haven't been flushed by the time ctx is cancelled are dropped — the agent
// records carrying those pending server messages remain in MongoDB and will
// be re-discovered the next time NotifyAgentUpdated fires for them.
//
// If workers fail to drain within shutdownCeiling (because a downstream call
// ignores ctx cancellation), Run logs a leak warning and returns anyway so
// the FX executor's OnStop hook isn't blocked.
func (s *AgentNotificationService) Run(ctx context.Context) error {
	for range s.workers {
		s.workerWG.Add(1)

		go s.dispatchWorker(ctx)
	}

	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.waitForWorkers()

			return nil
		case <-ticker.C:
			s.flushAll(ctx)
		case <-s.flushNow:
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

	s.enqueue(serverID, agent.Metadata.InstanceUID)

	return nil
}

// waitForWorkers closes the dispatch channel and waits for the worker pool to
// drain, with a hard ceiling to avoid blocking shutdown on a hung downstream.
//
// If the ceiling fires before workers drain, the wait-tracker goroutine is left
// parked on workerWG.Wait until those workers eventually return (or the process
// exits). This is bounded — the goroutine holds no resources beyond its stack
// and the closure references — and avoids the alternative of forcibly killing
// in-flight workers, which would corrupt the downstream call's state.
func (s *AgentNotificationService) waitForWorkers() {
	close(s.dispatchCh)

	done := make(chan struct{})

	go func() {
		s.workerWG.Wait()
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

func (s *AgentNotificationService) enqueue(serverID string, instanceUID uuid.UUID) {
	// Load first to avoid allocating a fresh bucket+set on every call. serverIDs
	// are stable (one per peer server, ~10 in production), so the key is almost
	// always present after warm-up.
	val, found := s.pending.Load(serverID)
	if !found {
		val, _ = s.pending.LoadOrStore(serverID, &pendingBucket{
			mu:   sync.Mutex{},
			uids: sets.NewUUID(),
		})
	}

	bucket, isBucket := val.(*pendingBucket)
	if !isBucket {
		s.logger.Error("unexpected pending bucket type", slog.String("serverID", serverID))

		return
	}

	bucket.mu.Lock()
	bucket.uids.Insert(instanceUID)
	size := bucket.uids.Len()
	bucket.mu.Unlock()

	if size >= s.maxBatchSize {
		// Non-blocking nudge: at most one pending flush signal at a time.
		select {
		case s.flushNow <- struct{}{}:
		default:
		}
	}
}

func (s *AgentNotificationService) flushAll(ctx context.Context) {
	sourceServerID := s.serverIdentityProvider.CurrentServerID()
	if sourceServerID == "" {
		sourceServerID = unknownServerID
	}

	type drained struct {
		targetServerID string
		uids           []uuid.UUID
	}

	var batches []drained

	s.pending.Range(func(key, val any) bool {
		targetServerID, keyOK := key.(string)
		if !keyOK {
			return true
		}

		bucket, valOK := val.(*pendingBucket)
		if !valOK {
			return true
		}

		bucket.mu.Lock()
		if bucket.uids.Len() == 0 {
			bucket.mu.Unlock()

			return true
		}

		uids := bucket.uids.List()
		bucket.uids = sets.NewUUID()
		bucket.mu.Unlock()

		batches = append(batches, drained{targetServerID: targetServerID, uids: uids})

		return true
	})

	for _, item := range batches {
		batch := notificationBatch{
			sourceServerID: sourceServerID,
			targetServerID: item.targetServerID,
			uids:           item.uids,
		}

		select {
		case s.dispatchCh <- batch:
		case <-ctx.Done():
			s.logger.Warn("dropping batch during flush: context done",
				slog.String("targetServerID", item.targetServerID),
				slog.Int("agentCount", len(item.uids)),
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

	// On shutdown the parent ctx is cancelled and the dispatch channel is still
	// being drained. Skip the downstream calls so they don't surface as flush
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
