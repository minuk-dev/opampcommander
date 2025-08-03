package clientutil

import (
	"context"
	"fmt"

	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	v1connection "github.com/minuk-dev/opampcommander/api/v1/connection"
	"github.com/minuk-dev/opampcommander/pkg/client"
)

const (
	// ChunkSize is the number of agents to fetch in each request.
	ChunkSize = 100
)

// ListAgentFullyFunc is a function type that takes a context and an agent, and returns an error.
type ListAgentFullyFunc func(ctx context.Context, agent []v1agent.Agent) error

// ListAgentFully lists all agents and applies the provided function to each agent.
// It continues to fetch agents until there are no more agents to fetch.
func ListAgentFully(ctx context.Context, cli *client.Client, agentFn ListAgentFullyFunc) error {
	// Initialize the continue token to an empty string
	continueToken := ""

	for {
		// List agents with the current continue token
		resp, err := cli.AgentService.ListAgents(
			ctx,
			client.WithContinueToken(continueToken),
			client.WithLimit(ChunkSize),
		)
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		// Iterate over each agent in the response
		if len(resp.Items) == 0 {
			return nil // No agents found, exit the loop
		}

		continueToken = resp.Metadata.Continue // Update the continue token for the next iteration

		err = agentFn(ctx, resp.Items)
		if err != nil {
			return fmt.Errorf("failed to apply function to agents: %w", err)
		}
	}
}

// ListConnectionFullyFunc is a function type that takes a context and a connection, and returns an error.
type ListConnectionFullyFunc func(ctx context.Context, connections []v1connection.Connection) error

// ListConnectionFully lists all connections and applies the provided function to each connection.
// It continues to fetch connections until there are no more connections to fetch.
func ListConnectionFully(ctx context.Context, cli *client.Client, connectionFn ListConnectionFullyFunc) error {
	// Initialize the continue token to an empty string
	continueToken := ""

	for {
		// List connections with the current continue token
		resp, err := cli.ConnectionService.ListConnections(
			ctx,
			client.WithContinueToken(continueToken),
			client.WithLimit(ChunkSize),
		)
		if err != nil {
			return fmt.Errorf("failed to list connections: %w", err)
		}

		// Iterate over each connection in the response
		if len(resp.Items) == 0 {
			return nil // No connections found, exit the loop
		}

		continueToken = resp.Metadata.Continue // Update the continue token for the next iteration

		err = connectionFn(ctx, resp.Items)
		if err != nil {
			return fmt.Errorf("failed to apply function to connections: %w", err)
		}
	}
}
