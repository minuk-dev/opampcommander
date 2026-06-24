package agentservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strconv"
	"sync/atomic"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model/vo"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

const (
	agentGroupServiceName = "AgentGroupService"
	// ChangedAgentGroupBufferSize is the buffer size for the changed agent group channel.
	ChangedAgentGroupBufferSize = 100
	// PropagationChunkSize is the number of agents to process in each batch when propagating changes.
	PropagationChunkSize = 50
	// DefaultReconcileInterval is how often the background loop re-scans every agent group
	// to repair any drift the event-driven path missed (e.g. messages dropped because the
	// buffered channel was full, or a SaveAgent that failed midway).
	DefaultReconcileInterval = 5 * time.Minute
	// DeletedGroupReconcileWindow is how long after deletion the reconcile loop keeps
	// re-processing a deleted group so its former members get the deleted group's config
	// dropped even if the delete-time propagation event was lost (full buffer, or a crash
	// between persisting the deletion and queuing it). Past this window the group's members
	// are assumed already cleared and re-scanning it would be wasted work, so it is skipped.
	// Sized above DefaultReconcileInterval so at least one reconcile tick falls inside it.
	DeletedGroupReconcileWindow = 3 * DefaultReconcileInterval
)

// ErrInvalidRemoteConfig is returned when inline remote config is missing required fields.
var ErrInvalidRemoteConfig = errors.New("invalid remote config: both spec and name are required for inline config")

// ErrDuplicateRemoteConfigName is returned when two of a group's remote configs resolve to
// the same name, which would otherwise silently drop one of them.
var ErrDuplicateRemoteConfigName = errors.New("duplicate remote config name within agent group")

var _ agentport.AgentGroupUsecase = (*AgentGroupService)(nil)
var _ agentport.AgentGroupRelatedUsecase = (*AgentGroupService)(nil)

// AgentGroupService is a struct that implements the AgentGroupUsecase interface.
type AgentGroupService struct {
	// main port
	persistencePort agentport.AgentGroupPersistencePort

	// related port
	remoteConfigPersistencePort agentport.AgentRemoteConfigPersistencePort
	certificatePersistencePort  agentport.CertificatePersistencePort

	// other domain usecases
	agentUsecase agentport.AgentUsecase

	// internalStatus
	changedAgentGroupCh chan *agentmodel.AgentGroup

	// utils
	clock  clock.Clock
	logger *slog.Logger
}

// NewAgentGroupService creates a new instance of AgentGroupService.
func NewAgentGroupService(
	persistencePort agentport.AgentGroupPersistencePort,
	agentRemoteConfigPersistencePort agentport.AgentRemoteConfigPersistencePort,
	certificatePersistencePort agentport.CertificatePersistencePort,
	agentUsecase agentport.AgentUsecase,
	logger *slog.Logger,
) *AgentGroupService {
	return &AgentGroupService{
		persistencePort:             persistencePort,
		remoteConfigPersistencePort: agentRemoteConfigPersistencePort,
		certificatePersistencePort:  certificatePersistencePort,
		agentUsecase:                agentUsecase,
		clock:                       clock.NewRealClock(),
		logger:                      logger,
		changedAgentGroupCh:         make(chan *agentmodel.AgentGroup, ChangedAgentGroupBufferSize),
	}
}

// SetClock overrides the clock used for condition timestamps. Intended for tests.
func (s *AgentGroupService) SetClock(c clock.Clock) {
	s.clock = c
}

// Name implements scheduler.Scheduler.
func (s *AgentGroupService) Name() string {
	return agentGroupServiceName
}

// Run implements scheduler.Scheduler.
//
// Reconciliation runs on its own goroutine so a long pass (full collection scan +
// per-group agent updates) never blocks the changedAgentGroupCh consumer below.
// An initial reconcile fires immediately so post-restart drift is repaired without
// waiting the full DefaultReconcileInterval.
func (s *AgentGroupService) Run(ctx context.Context) error {
	go s.runReconcileLoop(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case agentGroup := <-s.changedAgentGroupCh:
			err := s.updateAgentsByAgentGroup(ctx, agentGroup)
			if err != nil {
				s.logger.Error("failed to propagate agent group changes to agents",
					slog.String("agent_group", agentGroup.Metadata.Name),
					slog.String("error", err.Error()),
				)
			}
		}
	}
	// unreachable
}

// GetAgentGroup retrieves an agent group by its namespace and name.
func (s *AgentGroupService) GetAgentGroup(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.AgentGroup, error) {
	agentGroup, err := s.persistencePort.GetAgentGroup(ctx, namespace, name, options)
	if err != nil {
		return nil, fmt.Errorf("get agent group: %w", err)
	}

	return agentGroup, nil
}

// ReconcileAgentGroup re-applies the named agent group to its matching agents on demand.
// It loads the (possibly deleted) group and runs the same update the background loop does,
// so callers can force a refresh without mutating the group or waiting for the next tick.
func (s *AgentGroupService) ReconcileAgentGroup(ctx context.Context, namespace, name string) error {
	agentGroup, err := s.persistencePort.GetAgentGroup(ctx, namespace, name, &model.GetOptions{IncludeDeleted: true})
	if err != nil {
		return fmt.Errorf("get agent group: %w", err)
	}

	err = s.updateAgentsByAgentGroup(ctx, agentGroup)
	if err != nil {
		return fmt.Errorf("reconcile agent group %s/%s: %w", namespace, name, err)
	}

	return nil
}

// SaveAgentGroup saves the agent group.
func (s *AgentGroupService) SaveAgentGroup(
	ctx context.Context,
	namespace string,
	name string,
	agentGroup *agentmodel.AgentGroup,
) (*agentmodel.AgentGroup, error) {
	agentGroup, err := s.persistencePort.PutAgentGroup(ctx, namespace, name, agentGroup)
	if err != nil {
		return nil, fmt.Errorf("save agent group: %w", err)
	}

	err = s.propagateAgentGroupChangesToAgents(ctx, agentGroup)
	if err != nil {
		return nil, fmt.Errorf("propagate agent group changes to agents: %w", err)
	}

	return agentGroup, nil
}

// ListAgentGroups retrieves a list of agent groups with pagination options.
func (s *AgentGroupService) ListAgentGroups(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentGroup], error) {
	resp, err := s.persistencePort.ListAgentGroups(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("list agent groups: %w", err)
	}

	return resp, nil
}

// DeleteAgentGroup marks an agent group as deleted.
func (s *AgentGroupService) DeleteAgentGroup(
	ctx context.Context,
	namespace string,
	name string,
	deletedAt time.Time,
	deletedBy string,
) error {
	agentGroup, err := s.persistencePort.GetAgentGroup(ctx, namespace, name, nil)
	if err != nil {
		return fmt.Errorf("failed to get agent group: %w", err)
	}

	agentGroup.MarkDeleted(deletedAt, deletedBy)

	_, err = s.persistencePort.PutAgentGroup(ctx, namespace, name, agentGroup)
	if err != nil {
		return fmt.Errorf("failed to delete agent group: %w", err)
	}

	// Propagate the deletion so agents that matched this group have their remote config
	// recomputed (the union of the remaining non-deleted matching groups). Without this an
	// agent keeps the deleted group's config indefinitely: the reconcile loop and the event
	// path are group-driven, and a deleted group is otherwise never revisited. The deleted
	// group still carries its selector, so updateAgentsByAgentGroup can find its former
	// members and ApplyMatchingAgentGroupsToAgent drops the now-deleted group's contribution.
	//
	// Best-effort, non-blocking: a full buffer must not hang this request handler. The
	// reconcile loop re-processes recently-deleted groups (DeletedGroupReconcileWindow) as
	// the durable safety net, so a dropped event still self-heals.
	select {
	case s.changedAgentGroupCh <- agentGroup:
	default:
		s.logger.Warn("agent group deletion not queued (buffer full); reconcile will drain former members",
			slog.String("agent_group", name),
			slog.String("namespace", namespace),
		)
	}

	return nil
}

// ListAgentsByAgentGroup lists agents that belong to the specified agent group.
func (s *AgentGroupService) ListAgentsByAgentGroup(
	ctx context.Context,
	agentGroup *agentmodel.AgentGroup,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	agentSelector := agentGroup.Spec.Selector

	listResp, err := s.agentUsecase.ListAgentsBySelector(
		ctx,
		agentSelector,
		options,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents by agent group: %w", err)
	}

	return listResp, nil
}

// GetAgentGroupsForAgent retrieves all agent groups that match the agent's attributes.
func (s *AgentGroupService) GetAgentGroupsForAgent(
	ctx context.Context,
	agent *agentmodel.Agent,
) ([]*agentmodel.AgentGroup, error) {
	// Get all agent groups
	allGroups, err := s.persistencePort.ListAgentGroups(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent groups: %w", err)
	}

	// Filter groups that match the agent. An agent group only governs agents in its own
	// namespace, so groups from other namespaces are skipped even when their selector would
	// otherwise match — without this scoping a group in namespace "foo" would (incorrectly)
	// apply its remote config to an agent in "default".
	var matchingGroups []*agentmodel.AgentGroup

	for _, group := range allGroups.Items {
		if group.IsDeleted() {
			continue
		}

		if group.Metadata.Namespace != agent.Metadata.Namespace {
			continue
		}

		if matchesSelector(agent, group.Spec.Selector) {
			matchingGroups = append(matchingGroups, group)
		}
	}

	return matchingGroups, nil
}

// matchesSelector checks if an agent matches the given selector.
func matchesSelector(agent *agentmodel.Agent, selector agentmodel.AgentSelector) bool {
	// Check identifying attributes
	for key, value := range selector.IdentifyingAttributes {
		agentValue, ok := agent.Metadata.Description.IdentifyingAttributes[key]
		if !ok || agentValue != value {
			return false
		}
	}

	// Check non-identifying attributes
	for key, value := range selector.NonIdentifyingAttributes {
		agentValue, ok := agent.Metadata.Description.NonIdentifyingAttributes[key]
		if !ok || agentValue != value {
			return false
		}
	}

	return true
}

// PropagateAgentRemoteConfigChange queues propagation for every agent group in the
// namespace that references the named AgentRemoteConfig via AgentRemoteConfigRef. Inline
// configs are stored on the group itself and need no re-propagation when an external
// AgentRemoteConfig changes.
//
// Per-group queue failures are logged and the loop continues — the reconcile loop is
// the durable safety net, but stopping mid-list would leave later groups stale until
// the next tick for no good reason.
func (s *AgentGroupService) PropagateAgentRemoteConfigChange(
	ctx context.Context,
	namespace string,
	remoteConfigName string,
) error {
	groups, err := s.persistencePort.ListAgentGroups(ctx, nil)
	if err != nil {
		return fmt.Errorf("list agent groups for remote-config change: %w", err)
	}

	for _, group := range groups.Items {
		if group.IsDeleted() || group.Metadata.Namespace != namespace {
			continue
		}

		if !agentGroupReferencesRemoteConfig(group, remoteConfigName) {
			continue
		}

		err := s.propagateAgentGroupChangesToAgents(ctx, group)
		if err != nil {
			s.logger.Warn("failed to queue agent group propagation after remote config change",
				slog.String("agent_group", group.Metadata.Name),
				slog.String("namespace", group.Metadata.Namespace),
				slog.String("remote_config", remoteConfigName),
				slog.String("error", err.Error()),
			)
		}
	}

	return nil
}

// ApplyMatchingAgentGroupsToAgent computes the desired remote-config and connection
// state from the union of all matching, non-deleted agent groups and applies it to the
// agent in place. RemoteConfigs are REPLACED (not merged) so entries left behind by
// previously-matching groups are cleared. The caller is responsible for persisting.
func (s *AgentGroupService) ApplyMatchingAgentGroupsToAgent(
	ctx context.Context,
	agent *agentmodel.Agent,
) error {
	groups, err := s.GetAgentGroupsForAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("get agent groups for agent: %w", err)
	}

	desired := make(map[string]agentmodel.AgentConfigFile)

	for _, group := range groups {
		configs, err := s.collectGroupRemoteConfigs(ctx, group)
		if err != nil {
			// A single group with an invalid/unresolvable config must not block the
			// other matching groups from applying. The failure is surfaced on that
			// group's RemoteConfigApplied condition (see recordRemoteConfigCondition),
			// so skipping here is observable rather than silent.
			s.logger.Warn("skip agent group with unresolved remote config",
				slog.String("agent_group", group.Metadata.Name),
				slog.String("namespace", group.Metadata.Namespace),
				slog.String("error", err.Error()),
			)

			continue
		}

		maps.Copy(desired, configs)
	}

	setAgentRemoteConfigs(agent, desired)

	// Connection settings still follow per-group apply semantics (last group wins).
	// Multi-group connection conflicts remain a known limitation.
	for _, group := range groups {
		err := s.applyConnectionSettings(ctx, group, agent)
		if err != nil {
			return fmt.Errorf("apply connection settings from group %s: %w", group.Metadata.Name, err)
		}
	}

	return nil
}

// collectGroupRemoteConfigs resolves every remote config declared on the group into a
// flat name → file map without mutating any agent. ApplyMatchingAgentGroupsToAgent
// composes the result across all matching groups so an agent's spec reflects the
// current desired state — not the cumulative history of every group that ever matched.
func (s *AgentGroupService) collectGroupRemoteConfigs(
	ctx context.Context,
	group *agentmodel.AgentGroup,
) (map[string]agentmodel.AgentConfigFile, error) {
	out := make(map[string]agentmodel.AgentConfigFile)

	agentGroupName := group.Metadata.Name
	namespace := group.Metadata.Namespace

	for _, cfg := range group.Spec.AgentRemoteConfigs {
		file, name, err := s.resolveRemoteConfig(ctx, namespace, agentGroupName, cfg)
		if err != nil {
			return nil, err
		}

		// Two entries resolving to the same name with different content would silently
		// overwrite one another, dropping a config without any signal — surface that on
		// the group's RemoteConfigApplied condition. Identical duplicates are idempotent,
		// so collapse them instead of failing the whole group.
		if existing, dup := out[name]; dup {
			if !sameConfigFile(existing, file) {
				return nil, fmt.Errorf("%w: %q", ErrDuplicateRemoteConfigName, name)
			}

			continue
		}

		out[name] = file
	}

	return out, nil
}

// sameConfigFile reports whether two resolved config files are byte-for-byte equivalent,
// so idempotent duplicate entries can be collapsed rather than treated as a conflict.
func sameConfigFile(a, b agentmodel.AgentConfigFile) bool {
	return a.ContentType == b.ContentType && bytes.Equal(a.Body, b.Body)
}

// setAgentRemoteConfigs replaces the agent's spec.RemoteConfig with exactly the given
// set, nil-ing the field when the desired set is empty. This is what enables drop-on-
// reconcile semantics — any keys not present in `configs` are removed.
func setAgentRemoteConfigs(agent *agentmodel.Agent, configs map[string]agentmodel.AgentConfigFile) {
	if len(configs) == 0 {
		agent.Spec.RemoteConfig = nil

		return
	}

	agent.Spec.RemoteConfig = &agentmodel.AgentSpecRemoteConfig{
		ConfigMap: agentmodel.AgentConfigMap{
			ConfigMap: configs,
		},
	}
}

func (s *AgentGroupService) runReconcileLoop(ctx context.Context) {
	s.reconcileAll(ctx)

	ticker := time.NewTicker(DefaultReconcileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reconcileAll(ctx)
		}
	}
}

func (s *AgentGroupService) propagateAgentGroupChangesToAgents(
	ctx context.Context,
	agentGroup *agentmodel.AgentGroup,
) error {
	select {
	case s.changedAgentGroupCh <- agentGroup:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}
}

// reconcileAll re-scans every agent group and re-applies it to its matching agents.
// updateAgentsByAgentGroup is idempotent — it only writes when the agent's desired spec
// actually changed — so this is cheap when nothing has drifted. Recently-deleted groups
// are processed too (see shouldReconcileDeletedGroup) so a dropped delete event still
// results in the deleted group's config being dropped from its former members.
func (s *AgentGroupService) reconcileAll(ctx context.Context) {
	groups, err := s.persistencePort.ListAgentGroups(ctx, nil)
	if err != nil {
		s.logger.Error("reconcile loop: failed to list agent groups",
			slog.String("error", err.Error()))

		return
	}

	for _, group := range groups.Items {
		if group.IsDeleted() && !s.shouldReconcileDeletedGroup(group) {
			continue
		}

		err := s.updateAgentsByAgentGroup(ctx, group)
		if err != nil {
			s.logger.Warn("reconcile loop: failed to update agents for group",
				slog.String("agent_group", group.Metadata.Name),
				slog.String("namespace", group.Metadata.Namespace),
				slog.String("error", err.Error()),
			)
		}
	}
}

// shouldReconcileDeletedGroup reports whether a deleted group is still inside the window
// during which the reconcile loop keeps draining its former members (see
// DeletedGroupReconcileWindow). Groups missing a deletion timestamp are treated as in-window
// so they are not stranded with stale config on their members.
func (s *AgentGroupService) shouldReconcileDeletedGroup(group *agentmodel.AgentGroup) bool {
	deletedAt := group.GetDeletedAt()
	if deletedAt == nil {
		return true
	}

	return s.clock.Now().Sub(*deletedAt) <= DeletedGroupReconcileWindow
}

// agentSpecFingerprint hashes the parts of an agent's spec that are mutated by
// applyAgentGroupToAgent. The reconcile loop uses this to skip SaveAgent (and the
// cache write that follows) when applying a group leaves the agent unchanged.
//
// On marshalling failure the caller's `before == after` comparison must NOT skip the
// save, so we return a unique sentinel keyed on the agent identity — two unique
// sentinels never compare equal across calls for the same agent (they include a
// counter), forcing the save path on error.
func agentSpecFingerprint(agent *agentmodel.Agent) string {
	var connHash []byte
	if agent.Spec.ConnectionInfo != nil {
		connHash = agent.Spec.ConnectionInfo.Hash
	}

	payload := struct {
		RemoteConfig *agentmodel.AgentSpecRemoteConfig
		ConnHash     []byte
	}{
		RemoteConfig: agent.Spec.RemoteConfig,
		ConnHash:     connHash,
	}

	hash, err := vo.NewHashFromAny(payload)
	if err != nil {
		// Unique per call so before != after; the caller will fall through to SaveAgent.
		return "err:" + agent.Metadata.InstanceUID.String() + ":" + strconv.FormatInt(fingerprintErrSeq.Add(1), 10)
	}

	return hash.String()
}

// successive marshal failures so before != after in agentSpecFingerprint's callers;
// not a configuration knob.
//
//nolint:gochecknoglobals // process-wide error-path counter used only to differentiate
var fingerprintErrSeq atomic.Int64

// agentGroupReferencesRemoteConfig reports whether any of the group's remote configs
// references the named AgentRemoteConfig resource (i.e. via AgentRemoteConfigRef).
func agentGroupReferencesRemoteConfig(group *agentmodel.AgentGroup, name string) bool {
	for _, cfg := range group.Spec.AgentRemoteConfigs {
		if cfg.AgentRemoteConfigRef != nil && *cfg.AgentRemoteConfigRef == name {
			return true
		}
	}

	return false
}

// remoteConfigConditionReason identifies this service as the actor that records the
// RemoteConfigApplied condition on agent groups.
const remoteConfigConditionReason = agentGroupServiceName

// groupDeclaresRemoteConfig reports whether the group declares any remote config at all.
// Groups that declare none never get a RemoteConfigApplied condition — there is nothing
// to apply, so recording one would be noise.
func groupDeclaresRemoteConfig(group *agentmodel.AgentGroup) bool {
	return len(group.Spec.AgentRemoteConfigs) > 0
}

// recordRemoteConfigCondition resolves the group's declared remote configs and records the
// outcome on its RemoteConfigApplied condition so both failures and recoveries are visible
// through the API instead of only the server log. The condition is written onto a freshly
// re-read copy of the group — never onto the passed-in pointer, which is aliased to the one
// SaveAgentGroup hands back to the HTTP caller (writing it would be a data race on
// Status.Conditions) and may be a stale snapshot the reconcile loop read minutes ago
// (writing it would clobber a concurrent edit). The group is re-persisted only when the
// condition's status or message actually changed, so the periodic reconcile does not write
// on every tick. The resolution error (if any) is returned so callers can react.
func (s *AgentGroupService) recordRemoteConfigCondition(
	ctx context.Context,
	group *agentmodel.AgentGroup,
) error {
	// A deleted group is only processed to drain its former members; recording a condition
	// (and re-persisting it) on a tombstone would be misleading and pointless.
	if group.IsDeleted() || !groupDeclaresRemoteConfig(group) {
		return nil
	}

	_, resolveErr := s.collectGroupRemoteConfigs(ctx, group)

	status := model.ConditionStatusTrue
	message := "remote config resolved successfully"

	if resolveErr != nil {
		status = model.ConditionStatusFalse
		message = resolveErr.Error()
	}

	// Re-read the current persisted group so we mutate/persist a private copy rather than the
	// shared (and possibly stale) pointer. Failure to load is non-fatal: the resolve result
	// still flows back to the caller and the next reconcile retries the condition write.
	fresh, err := s.persistencePort.GetAgentGroup(ctx, group.Metadata.Namespace, group.Metadata.Name, nil)
	if err != nil {
		s.logger.Warn("failed to load agent group to record RemoteConfigApplied condition",
			slog.String("agent_group", group.Metadata.Name),
			slog.String("namespace", group.Metadata.Namespace),
			slog.String("error", err.Error()),
		)

		return resolveErr
	}

	// Skip the write when nothing changed to keep the reconcile loop idempotent.
	if existing := fresh.GetCondition(model.ConditionTypeRemoteConfigApplied); existing != nil &&
		existing.Status == status && existing.Message == message {
		return resolveErr
	}

	fresh.SetCondition(model.ConditionTypeRemoteConfigApplied, status,
		s.clock.Now(), remoteConfigConditionReason, message)

	_, putErr := s.persistencePort.PutAgentGroup(ctx, fresh.Metadata.Namespace, fresh.Metadata.Name, fresh)
	if putErr != nil {
		s.logger.Warn("failed to persist RemoteConfigApplied condition on agent group",
			slog.String("agent_group", fresh.Metadata.Name),
			slog.String("namespace", fresh.Metadata.Namespace),
			slog.String("error", putErr.Error()),
		)
	}

	return resolveErr
}

// recordAgentRemoteConfigCondition reflects the result of an agent group assigning a remote
// config to the agent onto the agent's RemoteConfigApplied condition, and reports whether the
// condition changed (so the caller folds it into the save decision). When a config is assigned
// to an agent that lacks the AcceptsRemoteConfig capability the condition is set to False with
// an explanatory message — that attempt is otherwise completely invisible because the config
// is silently never delivered. When no config is assigned the condition is left untouched.
func (s *AgentGroupService) recordAgentRemoteConfigCondition(
	agent *agentmodel.Agent,
	agentGroup *agentmodel.AgentGroup,
) bool {
	if !agent.HasAssignedRemoteConfig() {
		return false
	}

	status := agentmodel.AgentConditionStatusTrue
	message := fmt.Sprintf("remote config applied by agent group %q", agentGroup.Metadata.Name)

	if !agent.IsRemoteConfigSupported() {
		status = agentmodel.AgentConditionStatusFalse
		message = fmt.Sprintf(
			"agent group %q assigned a remote config but the agent does not accept remote config "+
				"(missing AcceptsRemoteConfig capability); it will not be delivered",
			agentGroup.Metadata.Name,
		)
	}

	prev := agent.GetCondition(agentmodel.AgentConditionTypeRemoteConfigApplied)

	agent.SetConditionAt(agentmodel.AgentConditionTypeRemoteConfigApplied, status,
		s.clock.Now(), agentGroupServiceName, message)

	return prev == nil || prev.Status != status || prev.Message != message
}

func (s *AgentGroupService) updateAgentsByAgentGroup(
	ctx context.Context,
	agentGroup *agentmodel.AgentGroup,
) error {
	// Resolve this group's config once up front and record the outcome on its condition.
	// This is what makes an invalid config (e.g. an inline config missing its name, or a
	// dangling AgentRemoteConfigRef) observable instead of failing silently per agent.
	_ = s.recordRemoteConfigCondition(ctx, agentGroup)

	var continueToken string

	for {
		agentsResp, err := s.ListAgentsByAgentGroup(ctx, agentGroup, &model.ListOptions{
			Limit:          PropagationChunkSize,
			Continue:       continueToken,
			IncludeDeleted: false,
		})
		if err != nil {
			return fmt.Errorf("list agents by agent group: %w", err)
		}

		if len(agentsResp.Items) == 0 {
			break
		}

		for _, agent := range agentsResp.Items {
			before := agentSpecFingerprint(agent)

			// Apply the full desired state (union of every matching group), not just
			// this group's contribution — otherwise we'd keep adding configs without
			// ever dropping ones a group removed.
			err := s.ApplyMatchingAgentGroupsToAgent(ctx, agent)
			if err != nil {
				return fmt.Errorf("apply matching groups to agent %s: %w", agent.Metadata.InstanceUID, err)
			}

			after := agentSpecFingerprint(agent)

			// Record on the agent whether the group-driven config could actually be applied.
			// Crucially this also flags agents that an agent group assigned a config to but
			// that cannot accept remote config — otherwise that attempt is invisible. The
			// condition can change even when the spec did not (e.g. capability flip), so it
			// participates in the save decision alongside the spec fingerprint.
			condChanged := s.recordAgentRemoteConfigCondition(agent, agentGroup)

			if before == after && !condChanged {
				continue
			}

			err = s.agentUsecase.SaveAgent(ctx, agent)
			if err != nil {
				return fmt.Errorf("save updated agent: %w", err)
			}
		}

		// No more pages to fetch
		if agentsResp.Continue == "" {
			break
		}

		continueToken = agentsResp.Continue
	}

	return nil
}

func (s *AgentGroupService) resolveRemoteConfig(
	ctx context.Context,
	namespace string,
	agentGroupName string,
	remoteConfig agentmodel.AgentGroupAgentRemoteConfig,
) (agentmodel.AgentConfigFile, string, error) {
	// Case 1: Reference to existing AgentRemoteConfig resource
	if remoteConfig.AgentRemoteConfigRef != nil {
		arc, err := s.remoteConfigPersistencePort.GetAgentRemoteConfig(
			ctx, namespace, *remoteConfig.AgentRemoteConfigRef, nil)
		if err != nil {
			return agentmodel.AgentConfigFile{}, "", fmt.Errorf("get agent remote config %s: %w",
				*remoteConfig.AgentRemoteConfigRef, err)
		}

		// Use the original resource name (no prefix needed for refs)
		return agentmodel.AgentConfigFile{
			Body:        arc.Spec.Value,
			ContentType: arc.Spec.ContentType,
		}, arc.Metadata.Name, nil
	}

	// Case 2: Inline/direct config definition. Name the offending group so an operator can
	// tell which entry broke — collectGroupRemoteConfigs applies a group's configs
	// atomically, so one invalid entry blocks the whole group.
	if remoteConfig.AgentRemoteConfigSpec == nil || remoteConfig.AgentRemoteConfigName == nil {
		return agentmodel.AgentConfigFile{}, "", fmt.Errorf("%w (agent group %q)", ErrInvalidRemoteConfig, agentGroupName)
	}

	// Prefix with AgentGroupName to avoid name collisions
	// Format: {AgentGroupName}/{AgentRemoteConfigName}
	prefixedName := fmt.Sprintf("%s/%s", agentGroupName, *remoteConfig.AgentRemoteConfigName)

	return agentmodel.AgentConfigFile{
		Body:        remoteConfig.AgentRemoteConfigSpec.Value,
		ContentType: remoteConfig.AgentRemoteConfigSpec.ContentType,
	}, prefixedName, nil
}

func (s *AgentGroupService) applyConnectionSettings(
	ctx context.Context,
	agentGroup *agentmodel.AgentGroup,
	agent *agentmodel.Agent,
) error {
	logger := s.logger.With(
		slog.String("agent.metadata.instanceUid", agent.Metadata.InstanceUID.String()),
		slog.String("agentgroup.metadata.name", agentGroup.Metadata.Name),
	)

	if !agentGroup.HasAgentConnectionConfig() {
		logger.Debug("skip to apply connection settings because agentGroup has no connection config")

		return nil
	}

	conn := agentGroup.Spec.AgentConnectionConfig
	if conn == nil {
		return nil
	}

	opampConnection := s.buildOpAMPConnection(
		ctx, agentGroup.Metadata.Namespace, conn.OpAMPConnection, logger,
	)
	ownMetrics := s.buildTelemetryConnection(
		ctx, agentGroup.Metadata.Namespace, conn.OwnMetrics, logger,
	)
	ownLogs := s.buildTelemetryConnection(
		ctx, agentGroup.Metadata.Namespace, conn.OwnLogs, logger,
	)
	ownTraces := s.buildTelemetryConnection(
		ctx, agentGroup.Metadata.Namespace, conn.OwnTraces, logger,
	)
	otherConnections := s.buildOtherConnections(
		ctx, agentGroup.Metadata.Namespace, conn.OtherConnections, logger,
	)

	err := agent.ApplyConnectionSettings(opampConnection, ownMetrics, ownLogs, ownTraces, otherConnections)
	if err != nil {
		return fmt.Errorf("apply connection settings: %w", err)
	}

	return nil
}

func (s *AgentGroupService) buildOpAMPConnection(
	ctx context.Context,
	namespace string,
	conn *agentmodel.OpAMPConnectionSettings,
	logger *slog.Logger,
) *agentmodel.AgentOpAMPConnectionSettings {
	if conn == nil {
		return nil
	}

	result := &agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: conn.DestinationEndpoint,
		Headers:             conn.Headers,
		Certificate:         nil,
	}

	if conn.CertificateName != nil {
		certificate, err := s.certificatePersistencePort.GetCertificate(ctx, namespace, *conn.CertificateName, nil)
		if err != nil {
			logger.Warn("failed to get certificate for OpAMP connection",
				slog.String("certificateName", *conn.CertificateName),
				slog.String("err", err.Error()),
			)
		} else {
			result.Certificate = certificate.ToAgentCertificate()
		}
	}

	return result
}

func (s *AgentGroupService) buildTelemetryConnection(
	ctx context.Context,
	namespace string,
	conn *agentmodel.TelemetryConnectionSettings,
	logger *slog.Logger,
) *agentmodel.AgentTelemetryConnectionSettings {
	if conn == nil {
		return nil
	}

	result := &agentmodel.AgentTelemetryConnectionSettings{
		DestinationEndpoint: conn.DestinationEndpoint,
		Headers:             conn.Headers,
		Certificate:         nil,
	}

	if conn.CertificateName != nil {
		certificate, err := s.certificatePersistencePort.GetCertificate(ctx, namespace, *conn.CertificateName, nil)
		if err != nil {
			logger.Warn("failed to get certificate for telemetry connection",
				slog.String("certificateName", *conn.CertificateName),
				slog.String("err", err.Error()),
			)

			return nil
		}

		result.Certificate = certificate.ToAgentCertificate()
	}

	return result
}

func (s *AgentGroupService) buildOtherConnections(
	ctx context.Context,
	namespace string,
	conns map[string]agentmodel.OtherConnectionSettings,
	logger *slog.Logger,
) map[string]agentmodel.AgentOtherConnectionSettings {
	return mapValuesWithFilterNil(conns,
		func(conn agentmodel.OtherConnectionSettings, _ string) *agentmodel.AgentOtherConnectionSettings {
			result := &agentmodel.AgentOtherConnectionSettings{
				DestinationEndpoint: conn.DestinationEndpoint,
				Headers:             conn.Headers,
				Certificate:         nil,
			}

			if conn.CertificateName != nil {
				certificate, err := s.certificatePersistencePort.GetCertificate(ctx, namespace, *conn.CertificateName, nil)
				if err != nil {
					logger.Warn("failed to get certificate for other connection",
						slog.String("certificateName", *conn.CertificateName),
						slog.String("err", err.Error()),
					)

					return nil
				}

				result.Certificate = certificate.ToAgentCertificate()
			}

			return result
		},
	)
}

func mapValuesWithFilterNil[K comparable, V, R any](in map[K]V, iteratee func(value V, key K) *R) map[K]R {
	result := make(map[K]R, len(in))

	for key, value := range in {
		transformed := iteratee(value, key)
		if transformed == nil {
			continue
		}

		result[key] = *transformed
	}

	return result
}
