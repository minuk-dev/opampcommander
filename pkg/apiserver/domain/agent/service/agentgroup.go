package agentservice

import (
	"bytes"
	"cmp"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/samber/lo"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model/vo"
	"github.com/minuk-dev/opampcommander/pkg/selector"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

const (
	agentGroupServiceName = "AgentGroupService"
	singleAgentCheckLimit = 2
	// ChangedAgentGroupBufferSize is the capacity of the group propagation Store.
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

	// leaderElector gates the periodic reconcile loop so only one node runs it.
	leaderElector agentport.LeaderElector

	// pending propagation operations
	changeStore agentport.AgentGroupChangeStore

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
	leaderElector agentport.LeaderElector,
	changeStore agentport.AgentGroupChangeStore,
	logger *slog.Logger,
) *AgentGroupService {
	return &AgentGroupService{
		persistencePort:             persistencePort,
		remoteConfigPersistencePort: agentRemoteConfigPersistencePort,
		certificatePersistencePort:  certificatePersistencePort,
		agentUsecase:                agentUsecase,
		leaderElector:               leaderElector,
		clock:                       clock.NewRealClock(),
		logger:                      logger,
		changeStore:                 changeStore,
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
// per-group agent updates) never blocks the change Store consumer below.
// An initial reconcile fires immediately so post-restart drift is repaired without
// waiting the full DefaultReconcileInterval.
func (s *AgentGroupService) Run(ctx context.Context) error {
	go s.runReconcileLoop(ctx)

	for ctx.Err() == nil {
		change, err := s.changeStore.Next(ctx)
		if err != nil {
			if ctx.Err() == nil {
				return fmt.Errorf("next agent group change: %w", err)
			}

			break
		}

		group, err := s.persistencePort.GetAgentGroup(ctx, change.Namespace, change.Name,
			&model.GetOptions{IncludeDeleted: true})
		if err == nil {
			// Use current configuration, but visit the agents affected when this
			// event was queued, even if the group's selector has since changed.
			group.Spec.Selector = change.Selector
			err = s.updateAgentsByAgentGroup(ctx, group)
		}

		if err != nil {
			s.logger.Error("failed to propagate agent group changes to agents",
				slog.String("agent_group", change.Name), slog.String("namespace", change.Namespace),
				slog.String("error", err.Error()))
		}
	}

	return nil
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
	if conn := agentGroup.Spec.AgentConnectionConfig; conn != nil && conn.OpAMPConnection != nil &&
		conn.OpAMPConnection.CertificateName != nil {
		members, err := s.ListAgentsByAgentGroup(ctx, agentGroup, &model.ListOptions{Limit: singleAgentCheckLimit})
		if err != nil {
			return nil, fmt.Errorf("list agents for OpAMP certificate: %w", err)
		}

		if len(members.Items) != 1 || members.RemainingItemCount != 0 {
			return nil, fmt.Errorf("%w: an OpAMP client certificate requires exactly one agent in the group",
				model.ErrInvalidArgument)
		}

		_, err = s.validatedOpAMPCertificate(ctx, namespace, *conn.OpAMPConnection.CertificateName,
			members.Items[0].Metadata.InstanceUID.String())
		if err != nil {
			return nil, err
		}
	}

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

// ListAgentGroupsForAgent delegates the paginated membership read to persistence.
func (s *AgentGroupService) ListAgentGroupsForAgent(
	ctx context.Context, agent *agentmodel.Agent, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentGroup], error) {
	response, err := s.persistencePort.ListAgentGroupsForAgent(ctx, agent, options)
	if err != nil {
		return nil, fmt.Errorf("list agent groups for agent: %w", err)
	}

	return response, nil
}

// ListAgentGroups retrieves a list of agent groups with pagination options.
func (s *AgentGroupService) ListAgentGroups(
	ctx context.Context,
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentGroup], error) {
	resp, err := s.persistencePort.ListAgentGroups(ctx, namespace, options)
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
	resourceVersion ...int64,
) error {
	agentGroup, err := s.persistencePort.GetAgentGroup(ctx, namespace, name, &model.GetOptions{IncludeDeleted: true})
	if err != nil {
		return fmt.Errorf("failed to get agent group: %w", err)
	}

	if len(resourceVersion) > 0 {
		if !agentGroup.Metadata.DeletedAt.IsZero() && resourceVersion[0] == agentGroup.Metadata.ResourceVersion-1 {
			return nil
		}

		err = model.CheckResourceVersion(resourceVersion[0], agentGroup.Metadata.ResourceVersion)
		if err != nil {
			return fmt.Errorf("resource precondition: %w", err)
		}
	}

	if !agentGroup.Metadata.DeletedAt.IsZero() {
		return nil
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
	if !s.changeStore.TryEnqueue(agentport.AgentGroupChange{
		Namespace: namespace, Name: name, Selector: agentGroup.Spec.Selector,
	}) {
		s.logger.Warn("agent group deletion not queued (buffer full); reconcile will drain former members",
			slog.String("agent_group", name), slog.String("namespace", namespace))
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

	if options == nil {
		options = &model.ListOptions{}
	}

	// A group's selector only applies inside its namespace.
	scoped := *options
	scoped.FieldSelector = append(append(selector.FieldSelector(nil), options.FieldSelector...), selector.FieldRequirement{
		Field: "metadata.namespace", Operator: selector.OpEquals, Value: agentGroup.Metadata.Namespace,
	})

	listResp, err := s.agentUsecase.ListAgentsBySelector(
		ctx,
		agentSelector,
		&scoped,
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
	// An agent group only governs agents in its own namespace, so the listing is
	// scoped to the agent's — without that scoping a group in namespace "foo"
	// would (incorrectly) apply its remote config to an agent in "default".
	//
	// An agent carrying no namespace is an error here rather than a cluster-wide
	// listing, which is what the port's refusal of an empty namespace buys: this
	// path fails loudly instead of applying every namespace's config to it.
	var matchingGroups []*agentmodel.AgentGroup

	var options *model.ListOptions

	for {
		groups, err := s.persistencePort.ListAgentGroups(ctx, agent.Metadata.Namespace, options)
		if err != nil {
			return nil, fmt.Errorf("failed to list agent groups: %w", err)
		}

		for _, group := range groups.Items {
			if !group.IsDeleted() && group.Spec.Selector.Matches(agent) {
				matchingGroups = append(matchingGroups, group)
			}
		}

		if groups.Continue == "" {
			break
		}

		//exhaustruct:ignore
		options = &model.ListOptions{Continue: groups.Continue}
	}

	slices.SortFunc(matchingGroups, func(a, b *agentmodel.AgentGroup) int {
		return cmp.Or(cmp.Compare(b.Spec.Priority, a.Spec.Priority), cmp.Compare(a.Metadata.Name, b.Metadata.Name))
	})

	return matchingGroups, nil
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
	groups, err := s.persistencePort.ListAgentGroups(ctx, namespace, nil)
	if err != nil {
		return fmt.Errorf("list agent groups for remote-config change: %w", err)
	}

	for _, group := range groups.Items {
		if group.IsDeleted() {
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
// agent in place. Each remote-config filename and the entire connection-settings bundle
// are won by the highest priority, then lexicographically smallest group name. RemoteConfig
// and ConnectionInfo are group-owned computed state: there is no independent per-agent
// configuration use case. Entries no longer supplied by matching groups are cleared.
// The caller is responsible for persisting.
func (s *AgentGroupService) ApplyMatchingAgentGroupsToAgent(
	ctx context.Context,
	agent *agentmodel.Agent,
) error {
	groups, err := s.GetAgentGroupsForAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("get agent groups for agent: %w", err)
	}

	desired := make(map[string]agentmodel.AgentConfigFile)

	// Copy lower-ranked groups first so higher-ranked groups overwrite filename conflicts.
	for _, group := range slices.Backward(groups) {
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

	var desiredConnection *agentmodel.ConnectionInfo

	if group, found := lo.Find(groups, (*agentmodel.AgentGroup).HasAgentConnectionConfig); found {
		// Resolve the winning bundle before mutating the agent. Certificate validation
		// failures must not retain a stale group offer or mix in another group's settings.
		desiredConnection, err = s.resolveConnectionSettings(ctx, group, agent)
		if err != nil {
			return fmt.Errorf("resolve connection settings from group %s: %w", group.Metadata.Name, err)
		}
	}

	setAgentRemoteConfigs(agent, desired)
	agent.Spec.ConnectionInfo = desiredConnection

	return nil
}

// ReconcileAgent re-applies the matching agent groups to the agent and persists the result.
// It mirrors the per-agent step of the background reconcile loop (apply then save), so an
// on-demand reconcile of a single agent actually takes effect — ApplyMatchingAgentGroupsToAgent
// alone only mutates the in-memory agent and leaves persistence to the caller.
func (s *AgentGroupService) ReconcileAgent(ctx context.Context, agent *agentmodel.Agent) error {
	before, beforeErr := agentSpecFingerprint(agent)

	err := s.ApplyMatchingAgentGroupsToAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("apply matching agent groups to agent %s: %w", agent.Metadata.InstanceUID, err)
	}

	after, afterErr := agentSpecFingerprint(agent)
	if beforeErr == nil && afterErr == nil && before == after {
		return nil
	}

	err = s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("save reconciled agent %s: %w", agent.Metadata.InstanceUID, err)
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
	s.reconcileAllIfLeader(ctx)

	ticker := time.NewTicker(DefaultReconcileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reconcileAllIfLeader(ctx)
		}
	}
}

// reconcileAllIfLeader runs the full reconcile pass only when this node is the elected
// leader, so an N-node deployment performs one reconcile per interval instead of N
// concurrent full scans (and the write contention they cause). The event-driven path
// (change Store) is deliberately not gated: it must run on whichever node served
// the change so propagation stays immediate.
//
// Leader determination fails open: if the elector errors, this node reconciles anyway
// so a transient server-registry read error cannot strand reconciliation cluster-wide.
func (s *AgentGroupService) reconcileAllIfLeader(ctx context.Context) {
	isLeader, err := s.leaderElector.IsLeader(ctx)
	if err != nil {
		s.logger.Warn("reconcile loop: leader election failed, reconciling anyway",
			slog.String("error", err.Error()))

		isLeader = true
	}

	if !isLeader {
		s.logger.Debug("reconcile loop: not the leader, skipping periodic reconcile")

		return
	}

	s.reconcileAll(ctx)
}

func (s *AgentGroupService) propagateAgentGroupChangesToAgents(
	ctx context.Context,
	agentGroup *agentmodel.AgentGroup,
) error {
	err := s.changeStore.Enqueue(ctx, agentport.AgentGroupChange{
		Namespace: agentGroup.Metadata.Namespace, Name: agentGroup.Metadata.Name, Selector: agentGroup.Spec.Selector,
	})
	if err != nil {
		return fmt.Errorf("enqueue agent group change: %w", err)
	}

	return nil
}

// reconcileAll repairs any drift between agent groups and agents. It runs two complementary
// passes because neither alone is sufficient:
//   - reconcileAllGroups (group-driven) iterates groups → their matching agents. It also
//     records the per-group and per-agent RemoteConfigApplied conditions and re-drains
//     recently-deleted groups.
//   - reconcileAllAgents (agent-centric) iterates every agent → the union of its matching
//     groups. This is the only pass that catches an agent that stopped matching a still-
//     existing group because its own identity attributes changed: from that group's point
//     of view the agent is simply no longer a member, so the group-driven pass never
//     revisits it to drop the group's now-stale contribution.
//
// Both passes are idempotent (they only write when an agent's desired spec actually
// changed), so running them back-to-back is cheap when nothing has drifted.
func (s *AgentGroupService) reconcileAll(ctx context.Context) {
	s.reconcileAllGroups(ctx)
	s.reconcileAllAgents(ctx)
}

// reconcileAllGroups re-scans every agent group and re-applies it to its matching agents.
// updateAgentsByAgentGroup is idempotent — it only writes when the agent's desired spec
// actually changed — so this is cheap when nothing has drifted. Recently-deleted groups
// are processed too (see shouldReconcileDeletedGroup) so a dropped delete event still
// results in the deleted group's config being dropped from its former members.
func (s *AgentGroupService) reconcileAllGroups(ctx context.Context) {
	groups, err := s.persistencePort.ListAllAgentGroups(ctx, nil)
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

// reconcileAllAgents re-applies the union of matching agent groups to every agent, dropping
// any config contributed by a group that no longer selects it. This is the agent-centric
// counterpart to reconcileAllGroups: because it starts from the agent (not the group), it
// self-heals the identity-change orphan case the group-driven pass structurally cannot — an
// agent whose attributes changed so a still-existing group's selector no longer matches it.
// An empty selector lists every agent across all namespaces; ApplyMatchingAgentGroupsToAgent
// re-scopes each agent to groups in its own namespace. Idempotent: an agent whose desired
// spec is unchanged is not written.
func (s *AgentGroupService) reconcileAllAgents(ctx context.Context) {
	var continueToken string

	// An empty selector (no attribute constraints) matches every agent.
	//exhaustruct:ignore
	allAgents := agentmodel.AgentSelector{}

	for {
		agentsResp, err := s.agentUsecase.ListAgentsBySelector(ctx, allAgents, &model.ListOptions{
			Limit:          PropagationChunkSize,
			Continue:       continueToken,
			IncludeDeleted: false,
		})
		if err != nil {
			s.logger.Error("reconcile loop: failed to list agents for agent-centric reconcile",
				slog.String("error", err.Error()))

			return
		}

		for _, agent := range agentsResp.Items {
			before, beforeErr := agentSpecFingerprint(agent)

			err := s.ApplyMatchingAgentGroupsToAgent(ctx, agent)
			if err != nil {
				s.logger.Warn("reconcile loop: failed to apply matching groups to agent",
					slog.String("agent", agent.Metadata.InstanceUID.String()),
					slog.String("error", err.Error()),
				)

				continue
			}

			after, afterErr := agentSpecFingerprint(agent)
			if beforeErr == nil && afterErr == nil && before == after {
				continue
			}

			err = s.agentUsecase.SaveAgent(ctx, agent)
			if err != nil {
				s.logger.Warn("reconcile loop: failed to save agent-centric reconciled agent",
					slog.String("agent", agent.Metadata.InstanceUID.String()),
					slog.String("error", err.Error()),
				)
			}
		}

		if agentsResp.Continue == "" {
			break
		}

		continueToken = agentsResp.Continue
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
// A hashing failure is returned so callers always take the save path on error.
func agentSpecFingerprint(agent *agentmodel.Agent) (string, error) {
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
		return "", fmt.Errorf("fingerprint agent spec: %w", err)
	}

	return hash.String(), nil
}

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
) bool {
	if !agent.HasAssignedRemoteConfig() {
		return false
	}

	status := agentmodel.AgentConditionStatusTrue
	message := "remote config assigned by matching agent groups"

	if !agent.IsRemoteConfigSupported() {
		status = agentmodel.AgentConditionStatusFalse
		message = "matching agent groups assigned a remote config but the agent does not accept remote config " +
			"(missing AcceptsRemoteConfig capability); it will not be delivered"
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
			before, beforeErr := agentSpecFingerprint(agent)

			// Apply the full desired state (union of every matching group), not just
			// this group's contribution — otherwise we'd keep adding configs without
			// ever dropping ones a group removed.
			err := s.ApplyMatchingAgentGroupsToAgent(ctx, agent)
			if err != nil {
				return fmt.Errorf("apply matching groups to agent %s: %w", agent.Metadata.InstanceUID, err)
			}

			after, afterErr := agentSpecFingerprint(agent)

			// Record on the agent whether the group-driven config could actually be applied.
			// Crucially this also flags agents that an agent group assigned a config to but
			// that cannot accept remote config — otherwise that attempt is invisible. The
			// condition can change even when the spec did not (e.g. capability flip), so it
			// participates in the save decision alongside the spec fingerprint.
			condChanged := s.recordAgentRemoteConfigCondition(agent)

			if beforeErr == nil && afterErr == nil && before == after && !condChanged {
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

func (s *AgentGroupService) resolveConnectionSettings(
	ctx context.Context,
	agentGroup *agentmodel.AgentGroup,
	agent *agentmodel.Agent,
) (*agentmodel.ConnectionInfo, error) {
	logger := s.logger.With(
		slog.String("agent.metadata.instanceUid", agent.Metadata.InstanceUID.String()),
		slog.String("agentgroup.metadata.name", agentGroup.Metadata.Name),
	)

	conn := agentGroup.Spec.AgentConnectionConfig
	if conn == nil {
		return nil, nil //nolint:nilnil // absent or unsafe group bundle
	}

	var opampConnection *agentmodel.AgentOpAMPConnectionSettings

	if conn.OpAMPConnection != nil {
		var err error

		opampConnection, err = s.buildOpAMPConnection(
			ctx, agentGroup.Metadata.Namespace, agent.Metadata.InstanceUID.String(), conn.OpAMPConnection,
		)
		if err != nil {
			logger.Warn("skip unsafe OpAMP connection settings", slog.String("error", err.Error()))

			return nil, nil //nolint:nilnil // absent or unsafe group bundle
		}
	}

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

	info, err := agentmodel.NewConnectionInfo(opampConnection, ownMetrics, ownLogs, ownTraces, otherConnections)
	if err != nil {
		return nil, fmt.Errorf("build connection settings: %w", err)
	}

	return info, nil
}

func (s *AgentGroupService) buildOpAMPConnection(
	ctx context.Context,
	namespace string,
	instanceUID string,
	conn *agentmodel.OpAMPConnectionSettings,
) (*agentmodel.AgentOpAMPConnectionSettings, error) {
	result := &agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: conn.DestinationEndpoint,
		Headers:             conn.Headers,
		Certificate:         nil,
	}

	if conn.CertificateName != nil {
		certificate, err := s.validatedOpAMPCertificate(ctx, namespace, *conn.CertificateName, instanceUID)
		if err != nil {
			return nil, err
		}

		result.Certificate = certificate.ToAgentCertificate()
	}

	return result, nil
}

func (s *AgentGroupService) validatedOpAMPCertificate(
	ctx context.Context, namespace, name, instanceUID string,
) (*agentmodel.Certificate, error) {
	certificate, err := s.certificatePersistencePort.GetCertificate(ctx, namespace, name, nil)
	if err != nil {
		return nil, fmt.Errorf("get OpAMP certificate %q: %w", name, err)
	}

	keyPair, err := tls.X509KeyPair(certificate.Spec.Cert, certificate.Spec.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: OpAMP certificate %q has an invalid key pair: %w", model.ErrInvalidArgument, name, err)
	}

	leaf, err := x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("%w: parse OpAMP certificate %q: %w", model.ErrInvalidArgument, name, err)
	}

	if leaf.Subject.CommonName != instanceUID {
		return nil, fmt.Errorf("%w: OpAMP certificate %q CN must equal agent instance UID %s",
			model.ErrInvalidArgument, name, instanceUID)
	}

	return certificate, nil
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
	result := make(map[string]agentmodel.AgentOtherConnectionSettings, len(conns))

	for name, conn := range conns {
		connection := agentmodel.AgentOtherConnectionSettings{
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

				continue
			}

			connection.Certificate = certificate.ToAgentCertificate()
		}

		result[name] = connection
	}

	return result
}
