package mongodb

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// An agent is "connected" when both hold, mirroring agentmodel.Agent.IsConnectedAt:
//
//   - status.connected is true (the explicit flag, flipped on WebSocket close), and
//   - status.lastCommunicatedAt is within the staleness window, i.e. strictly greater
//     than ($$NOW - DefaultConnectionStaleness). This catches HTTP-polling agents that
//     simply stopped polling: their flag stays true but their last-report timestamp
//     drifts past the window.
//
// Sharing this predicate across the per-agent Connected field (helper.Mapper),
// the agent-group connected/not-connected counts, and the ConnectedOnly list filter
// keeps all three consistent — previously group counts used the raw flag while the
// badge used staleness.
//
// CLOCK CAVEAT: the staleness boundary is evaluated against MongoDB's $$NOW (the
// server-side query start time), whereas helper.Mapper.MapAgentToAPI evaluates the
// per-agent Connected field against the apiserver process clock at response-encode
// time. The two clocks differ by the query→encode latency (and any apiserver↔Mongo
// NTP skew), so an agent whose lastCommunicatedAt sits within that sub-second delta
// of the 90s boundary can still be counted/filtered differently from its rendered
// badge. This narrows — but does not fully eliminate — the badge/count mismatch; a
// definitive fix would evaluate both against a single clock.

// connectedStalenessExpr is the aggregation boolean expression for the staleness half
// of "connected": lastCommunicatedAt is newer than the staleness cutoff. A missing/zero
// lastCommunicatedAt resolves to null, which is never $gt a real date, so it is treated
// as disconnected (matching IsConnectedAt's zero-time guard).
func connectedStalenessExpr() bson.M {
	// $$NOW is a Date and $subtract treats a numeric operand as milliseconds.
	stalenessMillis := agentmodel.DefaultConnectionStaleness.Milliseconds()

	return bson.M{"$gt": []any{
		"$status.lastCommunicatedAt",
		bson.M{"$subtract": []any{"$$NOW", stalenessMillis}},
	}}
}

// connectedAggExpr is the full "connected" boolean expression for use inside
// aggregation expression contexts (e.g. $cond accumulators).
func connectedAggExpr() bson.M {
	return bson.M{"$and": []any{
		bson.M{"$eq": []any{"$status.connected", true}},
		connectedStalenessExpr(),
	}}
}

// notConnectedAggExpr is the logical negation of connectedAggExpr.
func notConnectedAggExpr() bson.M {
	return bson.M{"$not": []any{connectedAggExpr()}}
}

// connectedMatchFilter returns a find()/$match filter selecting only connected agents.
// The status.connected equality is kept as a plain field match (rather than buried in
// $expr) so the status.connected index can prefilter; the date-dependent staleness
// half — which no plain index serves — is left to $expr and only evaluated on the
// already-narrowed connected documents.
func connectedMatchFilter() bson.M {
	return bson.M{
		"status.connected": true,
		"$expr":            connectedStalenessExpr(),
	}
}
