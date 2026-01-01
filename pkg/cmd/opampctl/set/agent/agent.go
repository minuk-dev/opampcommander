// Package agent provides the command to set agent configurations.
package agent

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

const (
	// MaxCompletionResults is the maximum number of completion results to return.
	MaxCompletionResults = 20
)

var (
	// ErrCommandExecutionFailed is returned when the command execution fails.
	ErrCommandExecutionFailed = errors.New("command execution failed")
	// ErrInvalidHeaderFormat is returned when header format is invalid.
	ErrInvalidHeaderFormat = errors.New("invalid header format (expected key=value)")
)

// CommandOptions contains the options for the set agent command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType     string
	newInstanceUID string

	// internal
	client *client.Client
}

// NewCommand creates a new set agent command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Set agent configurations",
		Long: `Set various configurations for agents.

Available subcommands:
  new-instance-uid       Set a new instance UID for an agent
  connection-settings    Set connection settings for an agent`,
	}

	// Add subcommands
	cmd.AddCommand(newNewInstanceUIDCommand(options))
	cmd.AddCommand(newConnectionSettingsCommand(options))

	return cmd
}

// newNewInstanceUIDCommand creates the new-instance-uid subcommand.
func newNewInstanceUIDCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "new-instance-uid [AGENT_INSTANCE_UID] [NEW_INSTANCE_UID]",
		Short: "Set a new instance UID for an agent",
		Long: `Set a new instance UID for an agent.

The agent will be notified of the new instance UID when it next connects to the server.

Examples:
  # Set a new instance UID for an agent
  opampctl set agent new-instance-uid 550e8400-e29b-41d4-a716-446655440000 550e8400-e29b-41d4-a716-446655440001

  # Set a new instance UID and output as JSON
  opampctl set agent new-instance-uid 550e8400-e29b-41d4-a716-446655440000 \
    550e8400-e29b-41d4-a716-446655440001 -o json`,
		Args:              cobra.ExactArgs(2), //nolint:mnd // exactly 2 args are required
		ValidArgsFunction: options.ValidArgsFunction,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse agent instance UID
			instanceUID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid agent instance UID: %w", err)
			}

			// Get new instance UID from args
			options.newInstanceUID = args[1]

			err = options.Prepare(cmd, args)
			if err != nil {
				return err
			}

			return options.setNewInstanceUID(cmd, instanceUID)
		},
	}

	cmd.Flags().StringVarP(&options.formatType, "output", "o", "yaml", "Output format (yaml|json|table)")

	return cmd
}

// Prepare prepares the command for execution.
func (opts *CommandOptions) Prepare(*cobra.Command, []string) error {
	client, err := clientutil.NewClient(opts.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	opts.client = client

	return nil
}

// ValidArgsFunction provides dynamic completion for agent instance UIDs.
// Only completes the first argument (agent instance UID).
func (opts *CommandOptions) ValidArgsFunction(
	cmd *cobra.Command, args []string, toComplete string,
) ([]string, cobra.ShellCompDirective) {
	// Only provide completion for the first argument (agent instance UID)
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cli, err := clientutil.NewClient(opts.GlobalConfig)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	agentService := cli.AgentService

	// Use search API with the toComplete string as query
	resp, err := agentService.SearchAgents(
		cmd.Context(),
		toComplete,
		client.WithLimit(MaxCompletionResults),
	)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	instanceUIDs := lo.Map(resp.Items, func(agent v1agent.Agent, _ int) string {
		return agent.Metadata.InstanceUID.String()
	})

	return instanceUIDs, cobra.ShellCompDirectiveNoFileComp
}

// setNewInstanceUID sets a new instance UID for an agent.
func (opts *CommandOptions) setNewInstanceUID(cmd *cobra.Command, instanceUID uuid.UUID) error {
	newUID, err := uuid.Parse(opts.newInstanceUID)
	if err != nil {
		return fmt.Errorf("invalid new instance UID: %w", err)
	}

	request := v1agent.SetNewInstanceUIDRequest{
		NewInstanceUID: newUID,
	}

	agent, err := opts.client.AgentService.SetAgentNewInstanceUID(cmd.Context(), instanceUID, request)
	if err != nil {
		return fmt.Errorf("failed to set new instance UID: %w", err)
	}

	formatType := formatter.FormatType(opts.formatType)

	err = formatter.Format(cmd.OutOrStdout(), agent, formatType)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

// ConnectionSettingsOptions contains options for the connection-settings command.
type ConnectionSettingsOptions struct {
	// OpAMP connection settings
	opampEndpoint string
	opampHeaders  []string

	// Own metrics connection settings
	metricsEndpoint string
	metricsHeaders  []string

	// Own logs connection settings
	logsEndpoint string

	logsHeaders []string

	// Own traces connection settings
	tracesEndpoint string
	tracesHeaders  []string

	// TLS certificate settings
	certFile   string
	keyFile    string
	caCertFile string
}

// newConnectionSettingsCommand creates the connection-settings subcommand.
//
//nolint:funlen
func newConnectionSettingsCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	connOpts := ConnectionSettingsOptions{}

	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "connection-settings [AGENT_INSTANCE_UID]",
		Short: "Set connection settings for an agent",
		Long: `Set connection settings for an agent.

This command allows you to configure various connection settings for an agent, including:
- OpAMP server connection
- Own metrics/logs/traces destinations
- TLS certificates for secure connections

Examples:
  # Set OpAMP connection endpoint
  opampctl set agent connection-settings 550e8400-e29b-41d4-a716-446655440000 \
    --opamp-endpoint https://opamp.example.com:4320

  # Set metrics endpoint with headers
  opampctl set agent connection-settings 550e8400-e29b-41d4-a716-446655440000 \
    --metrics-endpoint https://metrics.example.com:4318/v1/metrics \
    --metrics-header "Authorization=Bearer token123"

  # Set logs endpoint
  opampctl set agent connection-settings 550e8400-e29b-41d4-a716-446655440000 \
    --logs-endpoint https://logs.example.com:4318/v1/logs

  # Set traces endpoint  
  opampctl set agent connection-settings 550e8400-e29b-41d4-a716-446655440000 \
    --traces-endpoint https://traces.example.com:4318/v1/traces

  # Set multiple settings at once
  opampctl set agent connection-settings 550e8400-e29b-41d4-a716-446655440000 \
    --opamp-endpoint https://opamp.example.com:4320 \
    --metrics-endpoint https://metrics.example.com:4318/v1/metrics \
    --logs-endpoint https://logs.example.com:4318/v1/logs`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: options.ValidArgsFunction,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnectionSettings(cmd, args, options, connOpts)
		},
	}

	// OpAMP flags
	cmd.Flags().StringVar(&connOpts.opampEndpoint, "opamp-endpoint", "",
		"OpAMP server endpoint URL")
	cmd.Flags().StringSliceVar(&connOpts.opampHeaders, "opamp-header", []string{},
		"OpAMP connection headers in key=value format (can be specified multiple times)")

	// Metrics flags
	cmd.Flags().StringVar(&connOpts.metricsEndpoint, "metrics-endpoint", "",
		"Own metrics destination endpoint URL")
	cmd.Flags().StringSliceVar(&connOpts.metricsHeaders, "metrics-header", []string{},
		"Metrics connection headers in key=value format (can be specified multiple times)")

	// Logs flags
	cmd.Flags().StringVar(&connOpts.logsEndpoint, "logs-endpoint", "",
		"Own logs destination endpoint URL")
	cmd.Flags().StringSliceVar(&connOpts.logsHeaders, "logs-header", []string{},
		"Logs connection headers in key=value format (can be specified multiple times)")

	// Traces flags
	cmd.Flags().StringVar(&connOpts.tracesEndpoint, "traces-endpoint", "",
		"Own traces destination endpoint URL")
	cmd.Flags().StringSliceVar(&connOpts.tracesHeaders, "traces-header", []string{},
		"Traces connection headers in key=value format (can be specified multiple times)")

	// TLS certificate flags (shared across all connections)
	cmd.Flags().StringVar(&connOpts.certFile, "cert-file", "",
		"Path to TLS certificate file")
	cmd.Flags().StringVar(&connOpts.keyFile, "key-file", "",
		"Path to TLS private key file")
	cmd.Flags().StringVar(&connOpts.caCertFile, "ca-cert-file", "",
		"Path to CA certificate file")

	return cmd
}

// runConnectionSettings executes the connection-settings command.
func runConnectionSettings(
	cmd *cobra.Command,
	args []string,
	opts CommandOptions,
	connOpts ConnectionSettingsOptions,
) error {
	// Parse agent instance UID
	instanceUID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid agent instance UID: %w", err)
	}

	// Initialize client
	opts.client, err = clientutil.NewClient(opts.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Build connection settings request
	request, err := buildConnectionSettingsRequest(connOpts)
	if err != nil {
		return fmt.Errorf("failed to build connection settings request: %w", err)
	}

	// Set connection settings
	err = opts.client.AgentService.SetAgentConnectionSettings(cmd.Context(), instanceUID, request)
	if err != nil {
		return fmt.Errorf("failed to set connection settings: %w", err)
	}

	cmd.Println("Connection settings updated successfully")

	return nil
}

// buildConnectionSettingsRequest builds the connection settings request from command options.
//
//nolint:funlen
func buildConnectionSettingsRequest(opts ConnectionSettingsOptions) (v1agent.SetConnectionSettingsRequest, error) {
	//exhaustruct:ignore
	var request v1agent.SetConnectionSettingsRequest

	// Read TLS certificate files if provided
	var cert v1agent.TLSCertificate

	if opts.certFile != "" {
		certData, err := os.ReadFile(opts.certFile)
		if err != nil {
			return request, fmt.Errorf("failed to read certificate file: %w", err)
		}

		cert.Cert = string(certData)
	}

	if opts.keyFile != "" {
		keyData, err := os.ReadFile(opts.keyFile)
		if err != nil {
			return request, fmt.Errorf("failed to read key file: %w", err)
		}

		cert.PrivateKey = string(keyData)
	}

	if opts.caCertFile != "" {
		caCertData, err := os.ReadFile(opts.caCertFile)
		if err != nil {
			return request, fmt.Errorf("failed to read CA certificate file: %w", err)
		}

		cert.CaCert = string(caCertData)
	}

	// Build OpAMP connection settings
	if opts.opampEndpoint != "" {
		headers, err := parseHeaders(opts.opampHeaders)
		if err != nil {
			return request, fmt.Errorf("failed to parse OpAMP headers: %w", err)
		}

		request.ConnectionSettings.OpAMP = v1agent.OpAMPConnectionSettings{
			DestinationEndpoint: opts.opampEndpoint,
			Headers:             headers,
			Certificate:         cert,
		}
	}

	// Build metrics connection settings
	if opts.metricsEndpoint != "" {
		headers, err := parseHeaders(opts.metricsHeaders)
		if err != nil {
			return request, fmt.Errorf("failed to parse metrics headers: %w", err)
		}

		request.ConnectionSettings.OwnMetrics = v1agent.TelemetryConnectionSettings{
			DestinationEndpoint: opts.metricsEndpoint,
			Headers:             headers,
			Certificate:         cert,
		}
	}

	// Build logs connection settings
	if opts.logsEndpoint != "" {
		headers, err := parseHeaders(opts.logsHeaders)
		if err != nil {
			return request, fmt.Errorf("failed to parse logs headers: %w", err)
		}

		request.ConnectionSettings.OwnLogs = v1agent.TelemetryConnectionSettings{
			DestinationEndpoint: opts.logsEndpoint,
			Headers:             headers,
			Certificate:         cert,
		}
	}

	// Build traces connection settings
	if opts.tracesEndpoint != "" {
		headers, err := parseHeaders(opts.tracesHeaders)
		if err != nil {
			return request, fmt.Errorf("failed to parse traces headers: %w", err)
		}

		request.ConnectionSettings.OwnTraces = v1agent.TelemetryConnectionSettings{
			DestinationEndpoint: opts.tracesEndpoint,
			Headers:             headers,
			Certificate:         cert,
		}
	}

	return request, nil
}

// parseHeaders parses header strings in "key=value" format into a map.
func parseHeaders(headerStrs []string) (map[string][]string, error) {
	if len(headerStrs) == 0 {
		return make(map[string][]string), nil
	}

	headers := make(map[string][]string)

	for _, headerStr := range headerStrs {
		key, value, found := strings.Cut(headerStr, "=")
		if !found {
			return nil, fmt.Errorf("%w: %s", ErrInvalidHeaderFormat, headerStr)
		}

		headers[key] = append(headers[key], value)
	}

	return headers, nil
}
