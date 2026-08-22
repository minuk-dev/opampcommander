package agentport

import (
	"context"
	"time"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// ReceiveServerEventHandler is a function type for handling received server events.
type ReceiveServerEventHandler func(ctx context.Context, message *serverevent.Message) error

// TransactionPort runs a unit of work inside a storage transaction so that a
// multi-step domain operation (e.g. a namespace cascade delete) commits
// atomically or rolls back as a whole. Implementations live in the secondary
// persistence adapters; the in-memory adapter is non-transactional and simply
// runs the callback inline.
type TransactionPort interface {
	// WithinTransaction starts a transaction, invokes fn with a derived
	// context, and commits if fn returns nil. If fn returns an error, the
	// transaction is rolled back and the error is propagated to the caller.
	// fn may be invoked more than once if the driver retries on transient
	// errors, so it must be idempotent.
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// AgentPersistencePort is an interface that defines the methods for agent persistence.
type AgentPersistencePort interface {
	// GetAgent retrieves an agent by its instance UID.
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error)
	// PutAgent saves or updates an agent.
	PutAgent(ctx context.Context, agent *agentmodel.Agent) error
	// UpdateAgentLiveness writes only the agent's liveness fields, leaving the rest
	// of the stored document — and its resource version — untouched.
	//
	// It exists because those fields have to stay current for the store's own sake:
	// the connected-agent list filter and the agent-group connected counts are
	// evaluated inside the datastore, against the stored timestamp, where no
	// read-side overlay can reach them. A full document write on that cadence is
	// what this whole mechanism is trying to avoid, so the write-behind flush uses
	// this narrow one instead.
	//
	// Leaving the resource version alone is deliberate: liveness carries no
	// optimistic-concurrency meaning, and bumping it would make routine heartbeats
	// invalidate concurrent API writes.
	//
	// It returns [model.ErrResourceNotExist] when no such agent is stored.
	UpdateAgentLiveness(ctx context.Context, liveness *agentmodel.AgentLiveness) error
	// DeleteAgent permanently removes an agent by its instance UID.
	DeleteAgent(ctx context.Context, instanceUID uuid.UUID) error
	// ListAgents retrieves a list of agents filtered by namespace with pagination options.
	ListAgents(ctx context.Context, namespace string,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error)
	// ListAgentsBySelector retrieves a list of agents matching the given selector.
	ListAgentsBySelector(
		ctx context.Context,
		selector agentmodel.AgentSelector,
		options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.Agent], error)
	// SearchAgents searches agents by query filtered by namespace with pagination options.
	SearchAgents(ctx context.Context, namespace string, query string,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error)
}

// AgentLivenessPort is the driven port for the fast tier of agent state: the
// liveness fields a heartbeat refreshes every few seconds (see
// [agentmodel.AgentLiveness]).
//
// It is a performance accelerator, never a source of truth. Implementations are
// expected to be cheap and may lose data — the durable record of an agent still
// lives behind [AgentPersistencePort], and the next heartbeat rebuilds anything
// dropped here. Callers therefore treat its errors as non-fatal.
type AgentLivenessPort interface {
	// Touch records the agent's current observation and returns the record as the
	// store now holds it, including the write-through anchor the store preserved.
	//
	// It returns the record rather than nothing so the caller can decide whether a
	// durable write is due without a second read: this runs on every agent message,
	// where a read-then-write would double both the latency and the op count. It
	// also removes the read-modify-write window in which two servers observing the
	// same agent could lose each other's update.
	//
	// The caller does not own [agentmodel.AgentLiveness.DurableReportedAt] here —
	// whatever it passes is ignored, and only MarkPersisted moves it.
	Touch(ctx context.Context, liveness *agentmodel.AgentLiveness) (*agentmodel.AgentLiveness, error)
	// MarkPersisted anchors the write-through throttle: it records reportedAt as the
	// observation the durable store now holds, leaving every field of the live
	// observation untouched.
	//
	// The caller passes the observation timestamp it wrote, not the wall clock —
	// staleness is measured from what the store holds, not from when it was told.
	MarkPersisted(ctx context.Context, instanceUID uuid.UUID, reportedAt time.Time) error
	// Get returns the record held for instanceUID, or nil (with a nil error) when
	// none is held. A missing record is a normal outcome, not an error.
	Get(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.AgentLiveness, error)
	// GetMany returns the records held for the given instance UIDs, keyed by UID.
	// UIDs with no record are omitted from the result rather than reported.
	GetMany(ctx context.Context, instanceUIDs []uuid.UUID) (map[uuid.UUID]*agentmodel.AgentLiveness, error)
	// Delete drops the record held for instanceUID. Deleting an absent record
	// succeeds.
	Delete(ctx context.Context, instanceUID uuid.UUID) error
	// ListPendingWriteThrough returns the records that hold an observation newer
	// than their last write-through AND were last written through before
	// notPersistedSince — the agents whose durable document has fallen behind by
	// more than the caller is willing to tolerate. limit bounds the batch; a limit
	// of zero or less means unbounded.
	//
	// The cutoff is a parameter rather than a fixed "anything pending" rule so the
	// write-behind flush does not duplicate writes the per-message throttle is about
	// to make anyway: pass a cutoff at least as old as that throttle window and the
	// flush stays a safety net until the throttle stops firing.
	//
	// Records come back oldest-write-through-first, so a saturated batch drains the
	// agents closest to falling outside the staleness window before the rest.
	ListPendingWriteThrough(
		ctx context.Context,
		notPersistedSince time.Time,
		limit int,
	) ([]*agentmodel.AgentLiveness, error)
}

// AgentLivenessMetricsPort records what the liveness fast tier is buying.
//
// The point of the tier is a number an operator cannot otherwise see: how many
// database writes the fleet's heartbeats are no longer costing. Absorbed
// observations minus write-throughs is that number, and it is only visible from
// inside the decision, so the domain emits it and an adapter records it.
//
// Implementations must be cheap and non-blocking — this runs on every agent
// message — and must never fail: there is no no-op default, so a nil-safe no-op
// implementation is wired when metrics are disabled.
type AgentLivenessMetricsPort interface {
	// RecordHeartbeatAbsorbed counts an observation the fast tier took without a
	// database write. This is the saved write.
	RecordHeartbeatAbsorbed()
	// RecordWriteThrough counts an observation written to the durable store, by the
	// shape of the write. The two shapes cost very differently — a full document
	// rewrite bumps the resource version and carries the whole agent, a liveness
	// write touches four fields — so an operator watching the tier's effect needs
	// them apart.
	RecordWriteThrough(shape LivenessWriteShape)
	// RecordFallback counts one operation served by the fallback tier because the
	// shared one could not answer.
	RecordFallback(operation string)
	// RecordBreakerState records the circuit breaker's current state, as a name
	// suitable for a metric label.
	RecordBreakerState(state string)
}

// LivenessWriteShape distinguishes the two ways an observation reaches the durable
// store.
type LivenessWriteShape string

// Liveness write shapes.
const (
	// LivenessWriteShapeDocument is a full agent document write, made when a message
	// changed durable state or when the message-path throttle window elapsed.
	LivenessWriteShapeDocument LivenessWriteShape = "document"
	// LivenessWriteShapeLiveness is a liveness-only write made by the write-behind
	// flush. Steady-state heartbeat traffic should land almost entirely here.
	LivenessWriteShapeLiveness LivenessWriteShape = "liveness"
)

// ServerEventSenderPort is an interface that defines the methods for sending events to servers.
type ServerEventSenderPort interface {
	// SendMessageToServer sends a message to the specified server. The caller passes the
	// resolved server so transports needing its address (e.g. direct) do not re-read it.
	SendMessageToServer(ctx context.Context, server *agentmodel.Server, message serverevent.Message) error
}

// ServerEventReceiverPort is an interface that defines the methods for receiving events from servers.
type ServerEventReceiverPort interface {
	// StartReceiver starts receiving messages from servers using the provided handler.
	StartReceiver(ctx context.Context, handler ReceiveServerEventHandler) error
}

// NamespacePersistencePort is an interface that defines the methods for namespace persistence.
type NamespacePersistencePort interface {
	// GetNamespace retrieves a namespace by its name.
	GetNamespace(ctx context.Context, name string,
		options *model.GetOptions) (*agentmodel.Namespace, error)
	// PutNamespace saves or updates a namespace.
	PutNamespace(ctx context.Context,
		namespace *agentmodel.Namespace) (*agentmodel.Namespace, error)
	// ListNamespaces retrieves a list of namespaces with pagination options.
	ListNamespaces(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Namespace], error)
}

// AgentGroupPersistencePort is an interface that defines the methods for agent group persistence.
type AgentGroupPersistencePort interface {
	// GetAgentGroup retrieves an agent group by its namespace and name.
	GetAgentGroup(ctx context.Context, namespace string, name string,
		options *model.GetOptions) (*agentmodel.AgentGroup, error)
	// PutAgentGroup saves the agent group.
	PutAgentGroup(ctx context.Context, namespace string, name string,
		agentGroup *agentmodel.AgentGroup) (*agentmodel.AgentGroup, error)
	// ListAgentGroups retrieves a list of agent groups with pagination options.
	ListAgentGroups(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.AgentGroup], error)
}

// ServerPersistencePort is an interface that defines the methods for server persistence.
type ServerPersistencePort interface {
	// GetServer retrieves a server by its ID.
	GetServer(ctx context.Context, id string) (*agentmodel.Server, error)
	// PutServer saves or updates a server.
	PutServer(ctx context.Context, server *agentmodel.Server) error
	// ListServers retrieves a list of all servers.
	ListServers(ctx context.Context) ([]*agentmodel.Server, error)
}

// ServerConnectionPersistencePort persists per-server connection records so the cluster-wide
// connection view can be queried from any node.
//
// Liveness is decoupled from membership: a server refreshes a single heartbeat every cycle
// (O(1)), while connection records are written only when they change. A record is visible in
// the cluster view only while its owning server's heartbeat is fresh, so a crashed server's
// connections drop out without any per-record rewrite.
type ServerConnectionPersistencePort interface {
	// SyncServerConnections refreshes serverID's heartbeat to heartbeatAt and applies an
	// incremental change set: upserts the given records (keyed by connection UID) and deletes
	// the given UIDs. In steady state upserts and deletes are empty and only the heartbeat is
	// written. Called once per snapshot cycle by the owning server.
	SyncServerConnections(
		ctx context.Context,
		serverID string,
		heartbeatAt time.Time,
		upserts []*agentmodel.ServerConnection,
		deletes []uuid.UUID,
	) error
	// RemoveServer deletes serverID's heartbeat and all its connection records, dropping the
	// server out of the cluster view immediately (graceful shutdown, or self-cleanup on start).
	RemoveServer(ctx context.Context, serverID string) error
	// ListServerConnections lists connections owned by servers whose heartbeat is at or after
	// notBefore (stale/crashed servers excluded); a zero notBefore includes every server. A
	// non-empty serverID restricts the result to that one server; an empty serverID spans all.
	// Results are filtered by namespace.
	ListServerConnections(ctx context.Context, namespace string, serverID string, notBefore time.Time,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.ServerConnection], error)
}

// AgentPackagePersistencePort is an interface that defines the methods for agent package persistence.
type AgentPackagePersistencePort interface {
	// GetAgentPackage retrieves an agent package by its namespace and name.
	GetAgentPackage(ctx context.Context, namespace string,
		name string, options *model.GetOptions) (*agentmodel.AgentPackage, error)
	// PutAgentPackage saves or updates an agent package.
	PutAgentPackage(ctx context.Context,
		agentPackage *agentmodel.AgentPackage) (*agentmodel.AgentPackage, error)
	// ListAgentPackages retrieves a list of agent packages with pagination options.
	ListAgentPackages(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.AgentPackage], error)
}

// AgentRemoteConfigPersistencePort is an interface that defines the methods for agent remote config persistence.
type AgentRemoteConfigPersistencePort interface {
	// GetAgentRemoteConfig retrieves an agent remote config by its namespace and name.
	GetAgentRemoteConfig(ctx context.Context, namespace string,
		name string, options *model.GetOptions) (*agentmodel.AgentRemoteConfig, error)
	// PutAgentRemoteConfig saves or updates an agent remote config.
	PutAgentRemoteConfig(
		ctx context.Context,
		config *agentmodel.AgentRemoteConfig,
	) (*agentmodel.AgentRemoteConfig, error)
	// ListAgentRemoteConfigs retrieves a list of agent remote configs with pagination options.
	ListAgentRemoteConfigs(
		ctx context.Context,
		options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error)
}

// EndpointPersistencePort is an interface that defines the methods for endpoint persistence.
type EndpointPersistencePort interface {
	// GetEndpoint retrieves an endpoint by its namespace and name.
	GetEndpoint(ctx context.Context, namespace string,
		name string, options *model.GetOptions) (*agentmodel.Endpoint, error)
	// PutEndpoint saves or updates an endpoint.
	PutEndpoint(
		ctx context.Context,
		endpoint *agentmodel.Endpoint,
	) (*agentmodel.Endpoint, error)
	// ListEndpoints retrieves a list of endpoints filtered by namespace with pagination options.
	ListEndpoints(
		ctx context.Context,
		namespace string,
		options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.Endpoint], error)
}

// RemoteConfigSchemaPersistencePort defines the methods for remote config schema persistence.
type RemoteConfigSchemaPersistencePort interface {
	// GetRemoteConfigSchema retrieves a schema by its namespace and name.
	GetRemoteConfigSchema(ctx context.Context, namespace string,
		name string, options *model.GetOptions) (*agentmodel.RemoteConfigSchema, error)
	// PutRemoteConfigSchema saves or updates a schema.
	PutRemoteConfigSchema(
		ctx context.Context,
		schema *agentmodel.RemoteConfigSchema,
	) (*agentmodel.RemoteConfigSchema, error)
	// ListRemoteConfigSchemas retrieves a list of schemas filtered by namespace with pagination options.
	ListRemoteConfigSchemas(
		ctx context.Context,
		namespace string,
		options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.RemoteConfigSchema], error)
}

// EndpointMetricsQueryPort queries a metrics backend (a Prometheus-compatible
// store) for how much telemetry collectors are sending to an endpoint. It is an
// outbound port implemented by a metrics-backend adapter; the endpoint's
// EndpointMetricsQuery templates select and aggregate the relevant series.
type EndpointMetricsQueryPort interface {
	// QueryEndpointThroughput evaluates the endpoint's per-signal PromQL
	// templates over the given rate window at instant `at`, returning the
	// aggregated per-second send throughput. Signals without a configured
	// template come back with Measured=false. An endpoint whose MetricsQuery is
	// nil/empty yields a result with every signal unmeasured (and no backend
	// call).
	QueryEndpointThroughput(
		ctx context.Context,
		endpoint *agentmodel.Endpoint,
		window time.Duration,
		at time.Time,
	) (*agentmodel.EndpointThroughput, error)
}

// HostPersistencePort is an interface that defines the methods for host persistence.
type HostPersistencePort interface {
	// GetHost retrieves a host by its ID. It returns port.ErrResourceNotExist
	// when no host with the given ID exists.
	GetHost(ctx context.Context, id string) (*agentmodel.Host, error)
	// PutHost saves or updates a host.
	PutHost(ctx context.Context, host *agentmodel.Host) (*agentmodel.Host, error)
	// ListHosts retrieves a list of hosts with pagination options.
	ListHosts(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Host], error)
}

// ContainerPersistencePort is an interface that defines the methods for container persistence.
type ContainerPersistencePort interface {
	// GetContainer retrieves a container by its ID. It returns
	// port.ErrResourceNotExist when no container with the given ID exists.
	GetContainer(ctx context.Context, id string) (*agentmodel.Container, error)
	// PutContainer saves or updates a container.
	PutContainer(ctx context.Context, container *agentmodel.Container) (*agentmodel.Container, error)
	// ListContainers retrieves a list of containers with pagination options.
	ListContainers(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Container], error)
}

// CertificatePersistencePort is an interface that defines the methods for certificate config persistence.
type CertificatePersistencePort interface {
	GetCertificate(ctx context.Context, namespace string,
		name string, options *model.GetOptions) (*agentmodel.Certificate, error)
	PutCertificate(ctx context.Context,
		certificate *agentmodel.Certificate) (*agentmodel.Certificate, error)
	ListCertificate(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Certificate], error)
}
