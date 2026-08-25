// Package redis provides the shared Redis implementation of the agent liveness
// port.
//
// It exists so a fleet's heartbeat traffic lands in one place every server can
// read, instead of one node-local view per server. That removes the two costs of
// the node-local tier: an agent that reconnects to a different replica no longer
// resets its write-through window, and reads from any server reflect liveness
// observed by every server.
//
// It is a pure accelerator. Nothing here is a source of truth — records carry a
// TTL and are expected to be lost — and every operation is bounded by a short
// command timeout so a slow Redis degrades to the durable store rather than
// holding up agent message processing.
package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/samber/lo"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ agentport.AgentLivenessPort = (*Store)(nil)

// ErrNoEndpoints is returned when the store is built without a Redis address.
var ErrNoEndpoints = errors.New("no redis endpoints configured")

const (
	// DefaultKeyPrefix namespaces every key this adapter owns, so a Redis shared
	// with other workloads stays legible.
	DefaultKeyPrefix = "opampcommander:agentliveness:"
	// pendingSuffix names the sorted set indexing agents whose durable document is
	// behind, scored by the time they were last written through.
	//
	// The index exists because the write-behind flush must answer "which agents are
	// overdue?" without scanning every key: at fleet scale a SCAN per flush cycle
	// would cost more than the writes the fast tier is saving.
	pendingSuffix = "pending"

	fieldConnected       = "connected"
	fieldConnectionType  = "connectionType"
	fieldSequenceNum     = "sequenceNum"
	fieldLastReportedAt  = "lastReportedAt"
	fieldLastReportedTo  = "lastReportedTo"
	fieldDurableReported = "durableReportedAt"
)

// Config holds the store's connection and behaviour settings.
type Config struct {
	// Endpoints are the Redis addresses. One address is a single server; several
	// address a cluster. Combine with MasterName to address a Sentinel set.
	Endpoints []string
	// MasterName selects Sentinel mode and names the monitored master.
	MasterName string
	Username   string
	Password   string
	// DB is the logical database index. Ignored in cluster mode.
	DB int
	// DialTimeout bounds establishing a connection.
	DialTimeout time.Duration
	// CommandTimeout bounds a single operation. Every method applies it, so a
	// stalled Redis fails fast instead of blocking the caller.
	CommandTimeout time.Duration
	// TTL is how long a record survives without being refreshed. It must outlive
	// the connection-staleness window; validated in the configuration layer.
	TTL time.Duration
	// KeyPrefix namespaces every key this store owns. Give two deployments sharing
	// one Redis distinct prefixes so they do not read each other's agents. Empty
	// selects [DefaultKeyPrefix].
	KeyPrefix string
}

// Store is the shared Redis implementation of [agentport.AgentLivenessPort].
type Store struct {
	client         goredis.UniversalClient
	ttl            time.Duration
	commandTimeout time.Duration
	keyPrefix      string
	pendingKey     string
}

// New creates a Redis-backed liveness store.
//
// A single UniversalClient covers single-server, cluster, and Sentinel
// deployments: go-redis picks the topology from the endpoint count and master
// name, so the deployment shape is a configuration question rather than a code one.
func New(config Config) (*Store, error) {
	if len(config.Endpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	//exhaustruct:ignore // go-redis has ~40 optional fields; the rest keep their defaults
	client := goredis.NewUniversalClient(&goredis.UniversalOptions{
		Addrs:       config.Endpoints,
		MasterName:  config.MasterName,
		Username:    config.Username,
		Password:    config.Password,
		DB:          config.DB,
		DialTimeout: config.DialTimeout,
		// Bound the socket operations too, not just our own context: a connection
		// that goes silent must not hold a caller for longer than the command budget.
		ReadTimeout:  config.CommandTimeout,
		WriteTimeout: config.CommandTimeout,
	})

	return NewWithClient(client, config), nil
}

// NewWithClient builds a store over an existing client, for callers that own the
// connection (tests, or a client shared with another component).
func NewWithClient(client goredis.UniversalClient, config Config) *Store {
	prefix := config.KeyPrefix
	if prefix == "" {
		prefix = DefaultKeyPrefix
	}

	return &Store{
		client:         client,
		ttl:            config.TTL,
		commandTimeout: config.CommandTimeout,
		keyPrefix:      prefix,
		pendingKey:     prefix + pendingSuffix,
	}
}

// Ping checks that Redis is reachable.
func (s *Store) Ping(ctx context.Context) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	err := s.client.Ping(ctx).Err()
	if err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}

// Close releases the connection.
func (s *Store) Close() error {
	err := s.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close redis client: %w", err)
	}

	return nil
}

// Touch implements [agentport.AgentLivenessPort].
//
// One round trip: the observation fields, the TTL refresh, the pending-index entry
// and the read of the write-through anchor all go out together, and the anchor comes
// back in the same reply. The anchor is never written here, so an observation cannot
// reset the throttle window and two servers touching the same agent cannot lose each
// other's update.
//
// A plain pipeline, not a transaction: the record and the index live in different
// key slots, which a cluster transaction rejects outright. Nothing here needs
// atomicity — the tier is an accelerator whose worst case is a redundant write.
func (s *Store) Touch(
	ctx context.Context,
	liveness *agentmodel.AgentLiveness,
) (*agentmodel.AgentLiveness, error) {
	if liveness == nil {
		return nil, nil //nolint:nilnil // nothing to record and nothing to report
	}

	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	key := s.recordKey(liveness.InstanceUID)
	member := liveness.InstanceUID.String()

	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, encodeObservation(liveness))
	pipe.Expire(ctx, key, s.ttl)
	anchor := pipe.HGet(ctx, key, fieldDurableReported)
	// NX so the score — the write-through anchor — is only ever set by MarkPersisted.
	// An agent seen for the first time enters at zero, which reads as "never written
	// through" and so is always overdue.
	pipe.ZAddNX(ctx, s.pendingKey, goredis.Z{Score: 0, Member: member})

	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, goredis.Nil) {
		return nil, fmt.Errorf("failed to record agent liveness in redis: %w", err)
	}

	stored := liveness.Clone()
	stored.DurableReportedAt = timeFromNanos(parseInt(anchor.Val()))

	return stored, nil
}

// MarkPersisted implements [agentport.AgentLivenessPort].
//
// Writes the anchor field alone and re-scores the pending index, so it cannot
// clobber an observation another server recorded in between.
func (s *Store) MarkPersisted(ctx context.Context, instanceUID uuid.UUID, reportedAt time.Time) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	key := s.recordKey(instanceUID)

	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, fieldDurableReported, nanosOrZero(reportedAt))
	pipe.Expire(ctx, key, s.ttl)
	pipe.ZAdd(ctx, s.pendingKey, goredis.Z{
		Score:  float64(nanosOrZero(reportedAt)),
		Member: instanceUID.String(),
	})

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to anchor agent liveness in redis: %w", err)
	}

	return nil
}

// Get implements [agentport.AgentLivenessPort].
func (s *Store) Get(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.AgentLiveness, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	fields, err := s.client.HGetAll(ctx, s.recordKey(instanceUID)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read agent liveness from redis: %w", err)
	}

	return decodeRecord(instanceUID, fields), nil
}

// GetMany implements [agentport.AgentLivenessPort].
func (s *Store) GetMany(
	ctx context.Context,
	instanceUIDs []uuid.UUID,
) (map[uuid.UUID]*agentmodel.AgentLiveness, error) {
	if len(instanceUIDs) == 0 {
		return map[uuid.UUID]*agentmodel.AgentLiveness{}, nil
	}

	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	pipe := s.client.Pipeline()
	commands := lo.Map(instanceUIDs, func(instanceUID uuid.UUID, _ int) *goredis.MapStringStringCmd {
		return pipe.HGetAll(ctx, s.recordKey(instanceUID))
	})

	// A pipeline reports the first failing command as its error; the individual
	// results are still readable, so a partial answer is better than none here.
	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, goredis.Nil) {
		return nil, fmt.Errorf("failed to read agent liveness batch from redis: %w", err)
	}

	return lo.FilterSliceToMapI(instanceUIDs,
		func(instanceUID uuid.UUID, index int) (uuid.UUID, *agentmodel.AgentLiveness, bool) {
			record := decodeRecord(instanceUID, commands[index].Val())

			return instanceUID, record, record != nil
		}), nil
}

// Delete implements [agentport.AgentLivenessPort].
func (s *Store) Delete(ctx context.Context, instanceUID uuid.UUID) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	// A plain pipeline: the record and the index are in different key slots, which a
	// cluster transaction rejects.
	pipe := s.client.Pipeline()
	pipe.Del(ctx, s.recordKey(instanceUID))
	pipe.ZRem(ctx, s.pendingKey, instanceUID.String())

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete agent liveness from redis: %w", err)
	}

	return nil
}

// ListPendingWriteThrough implements [agentport.AgentLivenessPort].
//
// The index is scored by write-through time, so the cutoff is a range query and
// the oldest-first ordering the port promises comes for free.
func (s *Store) ListPendingWriteThrough(
	ctx context.Context,
	notPersistedSince time.Time,
	limit int,
) ([]*agentmodel.AgentLiveness, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	members, err := s.pendingMembers(ctx, notPersistedSince, limit)
	if err != nil {
		return nil, err
	}

	if len(members) == 0 {
		return nil, nil
	}

	return s.loadPending(ctx, members)
}

func (s *Store) pendingMembers(ctx context.Context, notPersistedSince time.Time, limit int) ([]string, error) {
	//exhaustruct:ignore // Offset/Count are only meaningful together with a limit
	rangeBy := &goredis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatInt(notPersistedSince.UnixNano(), 10),
	}

	if limit > 0 {
		rangeBy.Count = int64(limit)
	}

	members, err := s.client.ZRangeByScore(ctx, s.pendingKey, rangeBy).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read pending agent liveness index from redis: %w", err)
	}

	return members, nil
}

// loadPending resolves indexed members to records.
//
// Two kinds of member are filtered out here rather than by the index query. An entry
// whose record has expired is dropped from the index too — without that self-heal it
// would linger forever, since nothing else removes it. An entry whose record has
// nothing new to write stays in the index (the agent may report again at any moment)
// but is not returned, because the index is scored by write-through time alone and
// cannot tell a quiet agent from an overdue one.
func (s *Store) loadPending(ctx context.Context, members []string) ([]*agentmodel.AgentLiveness, error) {
	pipe := s.client.Pipeline()
	commands := make([]*goredis.MapStringStringCmd, 0, len(members))
	instanceUIDs := make([]uuid.UUID, 0, len(members))

	for _, member := range members {
		instanceUID, parseErr := uuid.Parse(member)
		if parseErr != nil {
			continue
		}

		instanceUIDs = append(instanceUIDs, instanceUID)
		commands = append(commands, pipe.HGetAll(ctx, s.recordKey(instanceUID)))
	}

	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, goredis.Nil) {
		return nil, fmt.Errorf("failed to read pending agent liveness from redis: %w", err)
	}

	records := make([]*agentmodel.AgentLiveness, 0, len(commands))
	stale := make([]any, 0)

	for i, command := range commands {
		record := decodeRecord(instanceUIDs[i], command.Val())
		if record == nil {
			stale = append(stale, instanceUIDs[i].String())

			continue
		}

		if !record.HasUnwrittenObservation() {
			continue
		}

		records = append(records, record)
	}

	if len(stale) > 0 {
		s.client.ZRem(ctx, s.pendingKey, stale...)
	}

	return records, nil
}

func (s *Store) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.commandTimeout <= 0 {
		return context.WithCancel(ctx)
	}

	return context.WithTimeout(ctx, s.commandTimeout)
}

func (s *Store) recordKey(instanceUID uuid.UUID) string {
	return s.keyPrefix + instanceUID.String()
}

// encodeObservation lists the fields an observation owns. The write-through anchor
// is deliberately absent: only MarkPersisted writes it, which is what keeps a
// heartbeat from resetting the throttle window.
//
// The connection type is stored as its name rather than its ordinal — matching the
// MongoDB adapter, keeping the hash legible to anyone inspecting Redis, and letting
// it decode through the domain's own round-trip instead of a numeric conversion.
func encodeObservation(liveness *agentmodel.AgentLiveness) []any {
	return []any{
		fieldConnected, boolToString(liveness.Connected),
		fieldConnectionType, liveness.ConnectionType.String(),
		fieldSequenceNum, liveness.SequenceNum,
		fieldLastReportedAt, liveness.LastReportedAt.UnixNano(),
		fieldLastReportedTo, liveness.LastReportedTo,
	}
}

// decodeRecord rebuilds a record from a hash, returning nil for an absent one — an
// expired or never-written key is a normal outcome, not an error.
//
// Every field is decoded into its own type rather than parsed as int64 and converted:
// a hash is external input — another deployment sharing the Redis, a stale key, a
// hand-edited value — and a narrowing conversion would silently truncate it. An
// unparseable field falls back to its zero value, which every consumer already
// treats as "not known".
func decodeRecord(instanceUID uuid.UUID, fields map[string]string) *agentmodel.AgentLiveness {
	if len(fields) == 0 {
		return nil
	}

	return &agentmodel.AgentLiveness{
		InstanceUID:       instanceUID,
		Connected:         fields[fieldConnected] == "1",
		ConnectionType:    agentmodel.ConnectionTypeFromString(fields[fieldConnectionType]),
		SequenceNum:       parseUint(fields[fieldSequenceNum]),
		LastReportedAt:    timeFromNanos(parseInt(fields[fieldLastReportedAt])),
		LastReportedTo:    fields[fieldLastReportedTo],
		DurableReportedAt: timeFromNanos(parseInt(fields[fieldDurableReported])),
	}
}

func boolToString(value bool) string {
	if value {
		return "1"
	}

	return "0"
}

// nanosOrZero keeps a zero time encoded as 0 rather than the nanosecond offset of
// the zero year, so it round-trips back to a zero time.
func nanosOrZero(at time.Time) int64 {
	if at.IsZero() {
		return 0
	}

	return at.UnixNano()
}

func timeFromNanos(nanos int64) time.Time {
	if nanos == 0 {
		return time.Time{}
	}

	return time.Unix(0, nanos)
}

// parseUint decodes an unsigned field. SequenceNum is a uint64, so it is parsed as
// one: routing it through int64 would reject anything above MaxInt64 and turn a
// negative value into a very large sequence number.
func parseUint(value string) uint64 {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}

	return parsed
}

func parseInt(value string) int64 {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}

	return parsed
}
