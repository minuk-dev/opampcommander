package clientutil

import (
	"context"
	"fmt"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
)

const (
	// ChunkSize is the number of agents to fetch in each request.
	ChunkSize = 100
)

// ListAgentFully lists all agents and applies the provided function to each agent.
// It continues to fetch agents until there are no more agents to fetch.
func ListAgentFully(ctx context.Context, cli *client.Client) ([]v1.Agent, error) {
	var agents []v1.Agent
	// Initialize the continue token to an empty string
	continueToken := ""

	for {
		opts := []client.ListOption{
			client.WithLimit(ChunkSize),
		}
		if continueToken != "" {
			opts = append(opts, client.WithContinueToken(continueToken))
		}
		// List agents with the current continue token
		resp, err := cli.AgentService.ListAgents(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to list agents: %w", err)
		}

		// Iterate over each agent in the response
		if len(resp.Items) == 0 {
			return agents, nil // No agents found, exit the loop
		}

		agents = append(agents, resp.Items...)
		continueToken = resp.Metadata.Continue // Update the continue token for the next iteration
	}
}

// ListAgentFullyByAgentGroup lists all agents in a specific agent group and applies
// the provided function to each agent.
// It continues to fetch agents until there are no more agents to fetch.
func ListAgentFullyByAgentGroup(
	ctx context.Context,
	cli *client.Client,
	agentGroupName string,
) ([]v1.Agent, error) {
	var agents []v1.Agent
	// Initialize the continue token to an empty string
	continueToken := ""

	for {
		opts := []client.ListOption{
			client.WithLimit(ChunkSize),
		}
		if continueToken != "" {
			opts = append(opts, client.WithContinueToken(continueToken))
		}
		// List agents with the current continue token
		resp, err := cli.AgentGroupService.ListAgentsByAgentGroup(ctx, agentGroupName, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to list agents by agent group: %w", err)
		}

		// Iterate over each agent in the response
		if len(resp.Items) == 0 {
			return agents, nil // No agents found, exit the loop
		}

		agents = append(agents, resp.Items...)
		continueToken = resp.Metadata.Continue // Update the continue token for the next iteration
	}
}

// ListConnectionFully lists all connections and applies the provided function to each connection.
// It continues to fetch connections until there are no more connections to fetch.
func ListConnectionFully(ctx context.Context, cli *client.Client) ([]v1.Connection, error) {
	var connections []v1.Connection
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
			return nil, fmt.Errorf("failed to list connections: %w", err)
		}

		// Iterate over each connection in the response
		if len(resp.Items) == 0 {
			return connections, nil // No connections found, exit the loop
		}

		connections = append(connections, resp.Items...)

		continueToken = resp.Metadata.Continue // Update the continue token for the next iteration
	}
}

// ListAgentGroupFully lists all agent groups and applies the provided function to each agent group.
// It continues to fetch agent groups until there are no more agent groups to fetch.
func ListAgentGroupFully(ctx context.Context, cli *client.Client) ([]v1.AgentGroup, error) {
	var agentGroups []v1.AgentGroup
	// Initialize the continue token to an empty string
	continueToken := ""

	for {
		// List agent groups with the current continue token
		resp, err := cli.AgentGroupService.ListAgentGroups(
			ctx,
			client.WithContinueToken(continueToken),
			client.WithLimit(ChunkSize),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to list agent groups: %w", err)
		}

		// Iterate over each agent group in the response
		if len(resp.Items) == 0 {
			return agentGroups, nil // No agent groups found, exit the loop
		}

		agentGroups = append(agentGroups, resp.Items...)

		continueToken = resp.Metadata.Continue // Update the continue token for the next iteration
	}
}
