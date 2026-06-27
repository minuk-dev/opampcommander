package agentport

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var (
	// ErrConnectionAlreadyExists is an error that indicates that the connection already exists.
	ErrConnectionAlreadyExists = errors.New("connection already exists")
	// ErrConnectionNotFound is an error that indicates that the connection was not found.
	ErrConnectionNotFound = errors.New("connection not found")
	// ErrAgentConnected indicates that a delete was attempted on a still-connected agent.
	// The connection guard is enforced here in the domain so it cannot be bypassed by
	// callers that hold an AgentUsecase directly.
	ErrAgentConnected = errors.New("agent is still connected; only disconnected agents can be deleted")
	// ErrNamespaceAlreadyExists indicates a namespace create was attempted for a name
	// that already exists.
	ErrNamespaceAlreadyExists = errors.New("namespace already exists")
	// ErrDefaultNamespaceUndeletable indicates a delete was attempted on the built-in
	// default namespace, which is protected.
	ErrDefaultNamespaceUndeletable = errors.New("default namespace cannot be deleted")
)

// AgentUsecase is an interface that defines the methods for agent use cases.
type AgentUsecase interface {
	// GetAgent retrieves an agent by its instance UID.
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error)
	// GetOrCreateAgent retrieves an agent by its instance UID or creates a new one if it does not exist.
	GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error)
	// ListAgentsBySelector lists agents by the given selector.
	ListAgentsBySelector(
		ctx context.Context,
		selector agentmodel.AgentSelector,
		options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.Agent], error)
	// SaveAgent saves the agent.
	SaveAgent(ctx context.Context, agent *agentmodel.Agent) error
	// DeleteAgent permanently (hard) removes a disconnected agent by its instance UID.
	// It enforces the "only disconnected agents may be deleted" policy and returns
	// ErrAgentConnected for a still-connected agent, so the guard cannot be bypassed.
	DeleteAgent(ctx context.Context, instanceUID uuid.UUID) error
	// ListAgents lists agents filtered by namespace.
	ListAgents(ctx context.Context, namespace string,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error)
	// SearchAgents searches agents by instance UID prefix filtered by namespace.
	SearchAgents(ctx context.Context, namespace string, query string,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error)
}

// AgentNotificationUsecase is an interface for notifying servers about agent changes.
type AgentNotificationUsecase interface {
	// NotifyAgentUpdated notifies the connected server that the agent has pending messages.
	NotifyAgentUpdated(ctx context.Context, agent *agentmodel.Agent) error
}

// AgentCacheInvalidationPublisher broadcasts agent cache invalidations to peer servers.
//
// It is used by external (API-driven) write paths so that a write made on one node is
// not served stale from another node's read cache. The heartbeat-driven persistence path
// deliberately does NOT use this — that volume would flood the cluster, and its cross-node
// staleness is bounded by the cache TTL and harmless.
type AgentCacheInvalidationPublisher interface {
	// BroadcastAgentCacheInvalidation asks every other alive server to drop the listed
	// agents from its cache. It is best-effort: delivery failures are logged, not returned
	// as hard errors, since the cache entry expires on its own within the TTL regardless.
	BroadcastAgentCacheInvalidation(ctx context.Context, instanceUIDs ...uuid.UUID) error
}

// AgentCacheInvalidator drops a single agent from the local in-process cache. It is the
// receiving end of [AgentCacheInvalidationPublisher]: a peer's broadcast resolves to this.
type AgentCacheInvalidator interface {
	// InvalidateCache removes the agent from the local cache, forcing the next read to
	// go to persistence.
	InvalidateCache(instanceUID uuid.UUID)
}

// NamespaceUsecase is an interface that defines the methods for namespace use cases.
type NamespaceUsecase interface {
	// GetNamespace retrieves a namespace by its name.
	GetNamespace(ctx context.Context, name string,
		options *model.GetOptions) (*agentmodel.Namespace, error)
	// ListNamespaces lists all namespaces.
	ListNamespaces(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Namespace], error)
	// SaveNamespace persists the namespace as-is without applying lifecycle rules.
	// It is the low-level write used by declarative bootstrap; application flows
	// should prefer CreateNamespace/UpdateNamespace.
	SaveNamespace(ctx context.Context,
		namespace *agentmodel.Namespace) (*agentmodel.Namespace, error)
	// CreateNamespace enforces name uniqueness, stamps the creation metadata
	// (timestamp + actor), and persists the namespace. It returns
	// ErrNamespaceAlreadyExists when a namespace with the same name exists.
	CreateNamespace(ctx context.Context, namespace *agentmodel.Namespace,
		actor string) (*agentmodel.Namespace, error)
	// UpdateNamespace loads the stored namespace, applies the mutable fields from
	// the supplied namespace while preserving immutable identity/lifecycle state,
	// and persists the result.
	UpdateNamespace(ctx context.Context, name string,
		namespace *agentmodel.Namespace) (*agentmodel.Namespace, error)
	// DeleteNamespace cascade-deletes the namespace's children (agent groups,
	// certificates, agent packages, agent remote configs) and the namespace itself
	// inside a single transaction. The built-in default namespace is protected and
	// returns ErrDefaultNamespaceUndeletable.
	DeleteNamespace(ctx context.Context, name string, actor string) error
}

// AgentPackageUsecase is an interface that defines the methods for agent package use cases.
type AgentPackageUsecase interface {
	// GetAgentPackage retrieves an agent package by its namespace and name.
	GetAgentPackage(ctx context.Context, namespace string,
		name string, options *model.GetOptions) (*agentmodel.AgentPackage, error)
	// ListAgentPackages lists all agent packages.
	ListAgentPackages(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.AgentPackage], error)
	// SaveAgentPackage persists the agent package as-is without applying lifecycle
	// rules. Application flows should prefer CreateAgentPackage/UpdateAgentPackage.
	SaveAgentPackage(ctx context.Context,
		agentPackage *agentmodel.AgentPackage) (*agentmodel.AgentPackage, error)
	// CreateAgentPackage stamps the creation metadata (timestamp + actor) and
	// persists the agent package.
	CreateAgentPackage(ctx context.Context, agentPackage *agentmodel.AgentPackage,
		actor string) (*agentmodel.AgentPackage, error)
	// UpdateAgentPackage loads the stored agent package, applies the mutable fields
	// from the supplied agent package while preserving immutable identity/lifecycle
	// state, and persists the result.
	UpdateAgentPackage(ctx context.Context, namespace string, name string,
		agentPackage *agentmodel.AgentPackage) (*agentmodel.AgentPackage, error)
	// DeleteAgentPackage deletes the agent package by its namespace and name.
	DeleteAgentPackage(ctx context.Context, namespace string, name string,
		deletedAt time.Time, deletedBy string) error
}

// AgentRemoteConfigUsecase is an interface that defines the methods for agent remote config use cases.
type AgentRemoteConfigUsecase interface {
	// GetAgentRemoteConfig retrieves an agent remote config by its namespace and name.
	GetAgentRemoteConfig(ctx context.Context, namespace string,
		name string, options *model.GetOptions) (*agentmodel.AgentRemoteConfig, error)
	// ListAgentRemoteConfigs lists all agent remote configs.
	ListAgentRemoteConfigs(
		ctx context.Context, options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error)
	// SaveAgentRemoteConfig persists the agent remote config as-is without applying
	// lifecycle rules. Application flows should prefer
	// CreateAgentRemoteConfig/UpdateAgentRemoteConfig.
	SaveAgentRemoteConfig(
		ctx context.Context, agentRemoteConfig *agentmodel.AgentRemoteConfig,
	) (*agentmodel.AgentRemoteConfig, error)
	// CreateAgentRemoteConfig stamps the creation metadata (timestamp + actor) and
	// persists the agent remote config.
	CreateAgentRemoteConfig(ctx context.Context, agentRemoteConfig *agentmodel.AgentRemoteConfig,
		actor string) (*agentmodel.AgentRemoteConfig, error)
	// UpdateAgentRemoteConfig loads the stored agent remote config, applies the
	// mutable fields from the supplied config while preserving immutable
	// identity/lifecycle state, and persists the result.
	UpdateAgentRemoteConfig(ctx context.Context, namespace string, name string,
		agentRemoteConfig *agentmodel.AgentRemoteConfig) (*agentmodel.AgentRemoteConfig, error)
	// DeleteAgentRemoteConfig deletes the agent remote config by its namespace and name.
	DeleteAgentRemoteConfig(ctx context.Context, namespace string, name string,
		deletedAt time.Time, deletedBy string) error
	// ReconcileAgentRemoteConfig re-runs the side effects normally triggered when the named
	// AgentRemoteConfig is created/updated: it detects telemetry endpoints from the config's
	// collector exporters and re-propagates the config to every agent group that references it.
	// Use this to repair drift for configs that predate those triggers (or were missed).
	ReconcileAgentRemoteConfig(ctx context.Context, namespace string, name string) error
}

// EndpointUsecase is an interface that defines the methods for endpoint use cases.
type EndpointUsecase interface {
	// GetEndpoint retrieves an endpoint by its namespace and name.
	GetEndpoint(ctx context.Context, namespace string,
		name string, options *model.GetOptions) (*agentmodel.Endpoint, error)
	// ListEndpoints lists endpoints filtered by namespace.
	ListEndpoints(
		ctx context.Context, namespace string, options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.Endpoint], error)
	// SaveEndpoint persists the endpoint as-is without applying lifecycle rules.
	// Application flows should prefer CreateEndpoint/UpdateEndpoint.
	SaveEndpoint(
		ctx context.Context, endpoint *agentmodel.Endpoint,
	) (*agentmodel.Endpoint, error)
	// CreateEndpoint validates the endpoint identity, rejects creating over an
	// existing endpoint, stamps the creation metadata (timestamp + actor), and
	// persists it. It returns ErrInvalidArgument for an empty name and
	// ErrResourceAlreadyExist when an endpoint with the same identity exists.
	CreateEndpoint(ctx context.Context, endpoint *agentmodel.Endpoint,
		actor string) (*agentmodel.Endpoint, error)
	// UpdateEndpoint loads the stored endpoint, applies the mutable fields from the
	// supplied endpoint while preserving immutable identity/lifecycle state, and
	// persists the result.
	UpdateEndpoint(ctx context.Context, namespace string, name string,
		endpoint *agentmodel.Endpoint) (*agentmodel.Endpoint, error)
	// DeleteEndpoint deletes the endpoint by its namespace and name.
	DeleteEndpoint(ctx context.Context, namespace string, name string,
		deletedAt time.Time, deletedBy string) error
}

// EndpointMetricsUsecase aggregates how much telemetry collectors are sending to
// endpoints by querying the metrics backend with each endpoint's configured query
// templates.
type EndpointMetricsUsecase interface {
	// GetEndpointThroughput aggregates the send throughput for a single endpoint,
	// evaluated over window at instant evaluatedAt.
	GetEndpointThroughput(ctx context.Context, namespace string, name string,
		window time.Duration, evaluatedAt time.Time) (*agentmodel.EndpointThroughput, error)
	// ListEndpointThroughput aggregates the send throughput for every endpoint in a
	// namespace, evaluated over window at instant evaluatedAt.
	ListEndpointThroughput(ctx context.Context, namespace string,
		window time.Duration, evaluatedAt time.Time) ([]*agentmodel.EndpointThroughput, error)
}

// EndpointDetectionUsecase detects telemetry backends from an AgentRemoteConfig's
// collector configuration and matches them to Endpoint resources.
type EndpointDetectionUsecase interface {
	// ReconcileEndpointsFromRemoteConfig matches every exporter destination in the
	// remote config to an endpoint (linking an existing same-URL endpoint or
	// auto-creating one). It never modifies a matched endpoint's spec and never
	// deletes endpoints.
	ReconcileEndpointsFromRemoteConfig(ctx context.Context, remoteConfig *agentmodel.AgentRemoteConfig) error
	// ExtractEndpointsFromAgent returns the endpoints an agent currently exports to,
	// parsed from its reported effective configuration. The returned endpoints are
	// ephemeral (not persisted) — a read-only view.
	ExtractEndpointsFromAgent(agent *agentmodel.Agent) ([]*agentmodel.Endpoint, error)
}

// AgentGroupUsecase is an interface that defines the methods for agent group use cases.
type AgentGroupUsecase interface {
	// GetAgentGroup retrieves an agent group by its namespace and name.
	GetAgentGroup(ctx context.Context, namespace string, name string,
		options *model.GetOptions) (*agentmodel.AgentGroup, error)
	// ListAgentGroups lists all agent groups.
	ListAgentGroups(
		ctx context.Context, options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.AgentGroup], error)
	// SaveAgentGroup saves the agent group.
	SaveAgentGroup(ctx context.Context, namespace string, name string,
		agentGroup *agentmodel.AgentGroup) (*agentmodel.AgentGroup, error)
	// DeleteAgentGroup deletes the agent group by its namespace and name.
	DeleteAgentGroup(ctx context.Context, namespace string, name string,
		deletedAt time.Time, deletedBy string) error
	// GetAgentGroupsForAgent retrieves all agent groups that match the agent's attributes.
	GetAgentGroupsForAgent(ctx context.Context, agent *agentmodel.Agent) ([]*agentmodel.AgentGroup, error)
	// PropagateAgentRemoteConfigChange re-applies all agent groups in the given namespace that
	// reference the named AgentRemoteConfig (via AgentRemoteConfigRef). Use this when the
	// AgentRemoteConfig resource itself changes — the agent group itself was not modified, so
	// the normal SaveAgentGroup-triggered propagation does not fire.
	PropagateAgentRemoteConfigChange(ctx context.Context, namespace, remoteConfigName string) error
	// ApplyMatchingAgentGroupsToAgent finds all agent groups that match the given agent and
	// applies their remote configs and connection settings. Use this when an agent reports a
	// new description so it picks up its assigned configs without waiting for a group update.
	// It mutates the agent in place; the caller is responsible for persisting.
	ApplyMatchingAgentGroupsToAgent(ctx context.Context, agent *agentmodel.Agent) error
	// ReconcileAgent re-applies the matching agent groups to the given agent and persists the
	// result. Unlike ApplyMatchingAgentGroupsToAgent it owns the save, so an on-demand
	// reconcile of a single agent actually takes effect.
	ReconcileAgent(ctx context.Context, agent *agentmodel.Agent) error
	// ReconcileAgentGroup re-applies the named agent group to its matching agents on demand,
	// the same work the background reconcile loop performs. Use this to force a refresh
	// without waiting for the next tick or mutating the group.
	ReconcileAgentGroup(ctx context.Context, namespace, name string) error
}

// AgentGroupRelatedUsecase is an interface that defines methods related to agent groups.
type AgentGroupRelatedUsecase interface {
	// ListAgentsByAgentGroup lists agents belonging to a specific agent group.
	ListAgentsByAgentGroup(
		ctx context.Context,
		agentGroup *agentmodel.AgentGroup,
		options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.Agent], error)
}

// HostUsecase is an interface that defines the methods for host use cases.
type HostUsecase interface {
	// GetHost retrieves a host by its ID.
	GetHost(ctx context.Context, id string) (*agentmodel.Host, error)
	// ListHosts lists all discovered hosts.
	ListHosts(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Host], error)
	// ObserveAgent upserts the host derived from an agent's reported description
	// and associates the agent with it. It is a no-op when the agent reports no
	// host attributes.
	ObserveAgent(ctx context.Context, agent *agentmodel.Agent) error
}

// ContainerUsecase is an interface that defines the methods for container use cases.
type ContainerUsecase interface {
	// GetContainer retrieves a container by its ID.
	GetContainer(ctx context.Context, id string) (*agentmodel.Container, error)
	// ListContainers lists all discovered containers.
	ListContainers(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Container], error)
	// ObserveAgent upserts the container derived from an agent's reported
	// description and associates the agent with it. It is a no-op when the agent
	// reports no container attributes.
	ObserveAgent(ctx context.Context, agent *agentmodel.Agent) error
}

// CertificateUsecase defines the interface for certificate use cases.
type CertificateUsecase interface {
	GetCertificate(ctx context.Context, namespace string,
		name string, options *model.GetOptions) (*agentmodel.Certificate, error)
	// SaveCertificate persists the certificate as-is without applying lifecycle
	// rules. Application flows should prefer CreateCertificate/UpdateCertificate.
	SaveCertificate(ctx context.Context,
		certificate *agentmodel.Certificate) (*agentmodel.Certificate, error)
	// CreateCertificate stamps the creation metadata (timestamp + actor) and
	// persists the certificate.
	CreateCertificate(ctx context.Context, certificate *agentmodel.Certificate,
		actor string) (*agentmodel.Certificate, error)
	// UpdateCertificate loads the stored certificate, applies the mutable fields
	// from the supplied certificate while preserving immutable identity/lifecycle
	// state, stamps the update, and persists the result.
	UpdateCertificate(ctx context.Context, namespace string, name string,
		certificate *agentmodel.Certificate, actor string) (*agentmodel.Certificate, error)
	ListCertificate(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Certificate], error)
	DeleteCertificate(ctx context.Context, namespace string, name string,
		deletedAt time.Time, deletedBy string) (*agentmodel.Certificate, error)
}

// ServerUsecase is an interface that defines the methods for server use cases.
type ServerUsecase interface {
	// ServerUsecase should also implement ServerMessageUsecase.
	ServerMessageUsecase
	// GetServer retrieves a server by its ID.
	GetServer(ctx context.Context, id string) (*agentmodel.Server, error)
	// ListServers lists all servers.
	ListServers(ctx context.Context) ([]*agentmodel.Server, error)
}

// ServerIdentityProvider is an interface that defines the methods for providing server identity.
type ServerIdentityProvider interface {
	// CurrentServer returns the current server.
	CurrentServer(ctx context.Context) (*agentmodel.Server, error)
	// CurrentServerID returns the identifier of the current server without performing I/O.
	CurrentServerID() string
}

// LeaderElector decides whether this server instance is the elected leader for
// cluster-singleton background work (currently the periodic agent-group reconcile),
// so that an N-node deployment runs such work once per interval instead of N times.
type LeaderElector interface {
	// IsLeader reports whether the current server is the leader. Callers should fail
	// open (run the work) when this returns an error, so a transient inability to
	// determine leadership never strands singleton work.
	IsLeader(ctx context.Context) (bool, error)
}

// ServerMessageUsecase is an interface that defines the methods for server message use cases.
type ServerMessageUsecase interface {
	// SendMessageToServerByServerID sends a message to the specified server.
	SendMessageToServerByServerID(ctx context.Context, serverID string, message serverevent.Message) error
	// SendMessageToServer sends a message to the specified server.
	SendMessageToServer(ctx context.Context, server *agentmodel.Server, message serverevent.Message) error
}

// ServerReceiverUsecase is an interface that defines the methods for server receiver use cases.
type ServerReceiverUsecase interface {
	// ReceiveMessageFromServer processes a message received from a server.
	ReceiveMessageFromServer() error
}

// ConnectionUsecase is an interface that defines the methods for connection use cases.
type ConnectionUsecase interface {
	// GetConnectionByInstanceUID returns the connection for the given instance UID.
	GetConnectionByInstanceUID(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Connection, error)
	// GetOrCreateConnectionByID returns the connection for the given ID or creates a new one.
	GetOrCreateConnectionByID(ctx context.Context, id any) (*agentmodel.Connection, error)
	// GetConnectionByID returns the connection for the given ID.
	GetConnectionByID(ctx context.Context, id any) (*agentmodel.Connection, error)
	// ListConnections returns the list of connections filtered by namespace. The result is
	// scoped to the current server instance (connections are live WebSockets that live on a
	// single node); in HA, query the agents API for a cluster-wide view of connectivity.
	ListConnections(ctx context.Context, namespace string,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Connection], error)
	// SaveConnection saves the connection.
	SaveConnection(ctx context.Context, connection *agentmodel.Connection) error
	// DeleteConnection deletes the connection.
	DeleteConnection(ctx context.Context, connection *agentmodel.Connection) error
	// SendServerToAgent sends a ServerToAgent message to the agent via WebSocket connection.
	SendServerToAgent(ctx context.Context, instanceUID uuid.UUID, message *protobufs.ServerToAgent) error
}

// ClusterConnectionUsecase exposes a cluster-wide view of connections, aggregated from the
// per-server snapshots each node periodically persists. Unlike ConnectionUsecase.ListConnections
// (which is node-local), this returns connections held by every alive server.
type ClusterConnectionUsecase interface {
	// ListClusterConnections returns connections filtered by namespace. A non-empty serverID
	// restricts the result to that one server; an empty serverID spans all servers. Records
	// from servers whose last snapshot is stale (e.g. a crashed node) are excluded.
	ListClusterConnections(ctx context.Context, namespace string, serverID string,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.ServerConnection], error)
}
