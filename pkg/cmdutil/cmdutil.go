// Package cmdutil contains utility functions for command-line commands.
package cmdutil

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
)

const (
	// MaxCompletionResults is the maximum number of completion results to return.
	MaxCompletionResults = 20
)

// AutoCompleteAgentInstanceUIDs provides auto-completion for agent instance UIDs.
func AutoCompleteAgentInstanceUIDs(
	ctx context.Context,
	agentService *client.AgentService,
	toComplete string,
) ([]string, error) {
	// Use search API with the toComplete string as query
	resp, err := agentService.SearchAgents(
		ctx,
		toComplete,
		client.WithLimit(MaxCompletionResults),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search agents for auto-completion: %w", err)
	}

	instanceUIDs := lo.Map(resp.Items, func(agent v1.Agent, _ int) string {
		return agent.Metadata.InstanceUID.String()
	})

	return instanceUIDs, nil
}
