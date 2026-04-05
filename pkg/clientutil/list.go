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

// ListAgentFully lists all agents in a namespace and applies the provided function to each agent.
// It continues to fetch agents until there are no more agents to fetch.
func ListAgentFully(ctx context.Context, cli *client.Client, namespace string) ([]v1.Agent, error) {
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
		resp, err := cli.AgentService.ListAgents(ctx, namespace, opts...)
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
	namespace string,
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
		resp, err := cli.AgentGroupService.ListAgentsByAgentGroup(ctx, namespace, agentGroupName, opts...)
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

// ListConnectionFully lists all connections in a namespace.
// It continues to fetch connections until there are no more connections to fetch.
func ListConnectionFully(
	ctx context.Context,
	cli *client.Client,
	namespace string,
) ([]v1.Connection, error) {
	var connections []v1.Connection
	// Initialize the continue token to an empty string
	continueToken := ""

	for {
		// List connections with the current continue token
		resp, err := cli.ConnectionService.ListConnections(
			ctx,
			namespace,
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
func ListAgentGroupFully(
	ctx context.Context,
	cli *client.Client,
	namespace string,
	opts ...client.ListOption,
) ([]v1.AgentGroup, error) {
	var agentGroups []v1.AgentGroup
	// Initialize the continue token to an empty string
	continueToken := ""

	for {
		// Build options for each request
		requestOpts := append([]client.ListOption{
			client.WithContinueToken(continueToken),
			client.WithLimit(ChunkSize),
		}, opts...)

		// List agent groups with the current continue token
		resp, err := cli.AgentGroupService.ListAgentGroups(ctx, namespace, requestOpts...)
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

// ListAgentPackageFully lists all agent packages and applies the provided function to each agent package.
// It continues to fetch agent packages until there are no more agent packages to fetch.
func ListAgentPackageFully(
	ctx context.Context,
	cli *client.Client,
	namespace string,
) ([]v1.AgentPackage, error) {
	var agentPackages []v1.AgentPackage
	// Initialize the continue token to an empty string
	continueToken := ""

	for {
		// List agent packages with the current continue token
		resp, err := cli.AgentPackageService.ListAgentPackages(
			ctx,
			namespace,
			client.WithContinueToken(continueToken),
			client.WithLimit(ChunkSize),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to list agent packages: %w", err)
		}

		// Iterate over each agent package in the response
		if len(resp.Items) == 0 {
			return agentPackages, nil // No agent packages found, exit the loop
		}

		agentPackages = append(agentPackages, resp.Items...)

		continueToken = resp.Metadata.Continue // Update the continue token for the next iteration
	}
}

// ListAgentRemoteConfigFully lists all agent remote configs in a namespace.
// It continues to fetch agent remote configs until there are no more to fetch.
func ListAgentRemoteConfigFully(
	ctx context.Context,
	cli *client.Client,
	namespace string,
) ([]v1.AgentRemoteConfig, error) {
	var agentRemoteConfigs []v1.AgentRemoteConfig
	// Initialize the continue token to an empty string
	continueToken := ""

	for {
		// List agent remote configs with the current continue token
		resp, err := cli.AgentRemoteConfigService.ListAgentRemoteConfigs(
			ctx,
			namespace,
			client.WithContinueToken(continueToken),
			client.WithLimit(ChunkSize),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to list agent remote configs: %w", err)
		}

		// Iterate over each agent remote config in the response
		if len(resp.Items) == 0 {
			return agentRemoteConfigs, nil // No agent remote configs found, exit the loop
		}

		agentRemoteConfigs = append(agentRemoteConfigs, resp.Items...)

		continueToken = resp.Metadata.Continue // Update the continue token for the next iteration
	}
}

// ListCertificateFully lists all certificates in a namespace.
// It continues to fetch certificates until there are no more to fetch.
func ListCertificateFully(
	ctx context.Context,
	cli *client.Client,
	namespace string,
) ([]v1.Certificate, error) {
	var certificates []v1.Certificate
	// Initialize the continue token to an empty string
	continueToken := ""

	for {
		// List certificates with the current continue token
		resp, err := cli.CertificateService.ListCertificates(
			ctx,
			namespace,
			client.WithContinueToken(continueToken),
			client.WithLimit(ChunkSize),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to list certificates: %w", err)
		}

		// Iterate over each certificate in the response
		if len(resp.Items) == 0 {
			return certificates, nil // No certificates found, exit the loop
		}

		certificates = append(certificates, resp.Items...)

		continueToken = resp.Metadata.Continue // Update the continue token for the next iteration
	}
}

// ListNamespaceFully lists all namespaces.
// It continues to fetch namespaces until there are no more namespaces to fetch.
func ListNamespaceFully(ctx context.Context, cli *client.Client) ([]v1.Namespace, error) {
	var namespaces []v1.Namespace

	continueToken := ""

	for {
		opts := []client.ListOption{
			client.WithLimit(ChunkSize),
		}
		if continueToken != "" {
			opts = append(opts, client.WithContinueToken(continueToken))
		}

		resp, err := cli.NamespaceService.ListNamespaces(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to list namespaces: %w", err)
		}

		if len(resp.Items) == 0 {
			return namespaces, nil
		}

		namespaces = append(namespaces, resp.Items...)
		continueToken = resp.Metadata.Continue
	}
}

// ListAcrossNamespaces lists resources across all namespaces by iterating
// over each namespace and calling the provided list function.
func ListAcrossNamespaces[T any](
	ctx context.Context,
	cli *client.Client,
	listFn func(ctx context.Context, namespace string) ([]T, error),
) ([]T, error) {
	namespaces, err := ListNamespaceFully(ctx, cli)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var result []T

	for _, namespace := range namespaces {
		items, err := listFn(ctx, namespace.Metadata.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to list resources in namespace %s: %w", namespace.Metadata.Name, err)
		}

		result = append(result, items...)
	}

	return result, nil
}

// ListUserFully lists all users.
// It continues to fetch users until there are no more users to fetch.
func ListUserFully(ctx context.Context, cli *client.Client) ([]v1.User, error) {
	var users []v1.User

	continueToken := ""

	for {
		opts := []client.ListOption{
			client.WithLimit(ChunkSize),
		}
		if continueToken != "" {
			opts = append(opts, client.WithContinueToken(continueToken))
		}

		resp, err := cli.UserService.ListUsers(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
		}

		if len(resp.Items) == 0 {
			return users, nil
		}

		users = append(users, resp.Items...)
		continueToken = resp.Metadata.Continue
	}
}

// ListRoleFully lists all roles.
// It continues to fetch roles until there are no more roles to fetch.
func ListRoleFully(ctx context.Context, cli *client.Client) ([]v1.Role, error) {
	var roles []v1.Role

	continueToken := ""

	for {
		opts := []client.ListOption{
			client.WithLimit(ChunkSize),
		}
		if continueToken != "" {
			opts = append(opts, client.WithContinueToken(continueToken))
		}

		resp, err := cli.RoleService.ListRoles(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to list roles: %w", err)
		}

		if len(resp.Items) == 0 {
			return roles, nil
		}

		roles = append(roles, resp.Items...)
		continueToken = resp.Metadata.Continue
	}
}
