// Package host provides the command to get host information.
package host

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// ErrCommandExecutionFailed is returned when the command execution fails.
var ErrCommandExecutionFailed = errors.New("command execution failed")

// CommandOptions contains the options for the host command.
type CommandOptions struct {
	*config.GlobalConfig

	formatType string
	client     *client.Client
}

// NewCommand creates a new host command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "host",
		Short: "get discovered hosts",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := options.Prepare(cmd, args)
			if err != nil {
				return err
			}

			return options.Run(cmd, args)
		},
	}
	cmd.Flags().StringVarP(
		&options.formatType, "output", "o", "short",
		"Output format (short, text, json, yaml)",
	)

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(_ *cobra.Command, _ []string) error {
	cli, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = cli

	return nil
}

// ItemForCLI is a struct for host display.
type ItemForCLI struct {
	ID       string `short:"ID"        text:"ID"`
	Name     string `short:"Name"      text:"Name"`
	Platform string `short:"Platform"  text:"Platform"`
	Agents   string `short:"Agents"    text:"Agents"`
	LastSeen string `short:"Last Seen" text:"Last Seen"`
}

func toItem(host v1.Host) ItemForCLI {
	return ItemForCLI{
		ID:       host.Metadata.ID,
		Name:     host.Metadata.Name,
		Platform: host.Spec.Platform,
		Agents:   strconv.Itoa(len(host.Status.AgentInstanceUIDs)),
		LastSeen: host.Metadata.LastSeenAt.String(),
	}
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return opt.Get(cmd, args)
	}

	return opt.List(cmd)
}

// List retrieves the list of hosts.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	resp, err := opt.client.HostService.ListHosts(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to list hosts: %w", err)
	}

	return opt.format(cmd, resp.Items)
}

// Get retrieves host(s) by ID.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	hosts := make([]v1.Host, 0, len(ids))

	for _, id := range ids {
		result, err := opt.client.HostService.GetHost(cmd.Context(), id)
		if err != nil {
			cmd.PrintErrf("failed to get host %s: %v\n", id, err)

			continue
		}

		hosts = append(hosts, *result)
	}

	if len(hosts) == 0 {
		cmd.Println("No hosts found.")

		return nil
	}

	return opt.format(cmd, hosts)
}

func (opt *CommandOptions) format(cmd *cobra.Command, hosts []v1.Host) error {
	formatType := formatter.FormatType(opt.formatType)

	var err error

	switch formatType {
	case formatter.SHORT, formatter.TEXT:
		items := lo.Map(hosts, func(item v1.Host, _ int) ItemForCLI {
			return toItem(item)
		})
		err = formatter.Format(cmd.OutOrStdout(), items, formatType)
	case formatter.JSON, formatter.YAML:
		err = formatter.Format(cmd.OutOrStdout(), hosts, formatType)
	default:
		return fmt.Errorf("unsupported format type: %s, %w", opt.formatType, ErrCommandExecutionFailed)
	}

	if err != nil {
		return fmt.Errorf("failed to format hosts: %w", err)
	}

	return nil
}
