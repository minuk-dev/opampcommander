package mongodb

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
)

// connectedAggExpr returns the aggregation boolean expression that decides whether
// an agent document is "connected". It mirrors agentmodel.Agent.IsConnectedAt:
//
//   - status.connected must be true (the explicit flag, flipped on WebSocket close), and
//   - status.lastCommunicatedAt must be within the staleness window, i.e. strictly
//     greater than ($$NOW - DefaultConnectionStaleness). This catches HTTP-polling
//     agents that simply stopped polling: their flag stays true but their last-report
//     timestamp drifts past the window.
//
// Using the same predicate for the per-agent Connected field (see
// helper.Mapper.MapAgentToAPI), the agent-group connected/not-connected counts, and
// the ConnectedOnly list filter keeps all three consistent — a previously divergent
// behaviour where group counts used the raw flag while the badge used staleness.
//
// $$NOW is the server-side query start time; a missing/zero lastCommunicatedAt
// resolves to null which is never $gt a real date, so it is treated as disconnected
// (matching IsConnectedAt's zero-time guard).
func connectedAggExpr() bson.M {
	// $$NOW is a Date and $subtract treats a numeric operand as milliseconds.
	stalenessMillis := agentmodel.DefaultConnectionStaleness.Milliseconds()

	return bson.M{"$and": []any{
		bson.M{"$eq": []any{"$status.connected", true}},
		bson.M{"$gt": []any{
			"$status.lastCommunicatedAt",
			bson.M{"$subtract": []any{"$$NOW", stalenessMillis}},
		}},
	}}
}

// notConnectedAggExpr is the logical negation of connectedAggExpr.
func notConnectedAggExpr() bson.M {
	return bson.M{"$not": []any{connectedAggExpr()}}
}

// connectedMatchFilter returns a find()/$match filter selecting only connected
// agents, reusing connectedAggExpr via $expr so the list filter and the aggregation
// counts can never drift apart.
func connectedMatchFilter() bson.M {
	return bson.M{"$expr": connectedAggExpr()}
}
