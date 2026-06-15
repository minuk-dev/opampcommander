// Package container provides the command to get container information.
package container

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

// CommandOptions contains the options for the container command.
type CommandOptions struct {
	*config.GlobalConfig

	formatType string
	client     *client.Client
}

// NewCommand creates a new container command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "container",
		Short: "get discovered containers",
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

// ItemForCLI is a struct for container display.
type ItemForCLI struct {
	ID       string `short:"ID"       text:"ID"`
	Name     string `short:"Name"     text:"Name"`
	Platform string `short:"Platform" text:"Platform"`
	Image    string `short:"Image"    text:"Image"`
	HostID   string `short:"Host"     text:"Host"`
	Agents   string `short:"Agents"   text:"Agents"`
}

func toItem(container v1.Container) ItemForCLI {
	return ItemForCLI{
		ID:       container.Metadata.ID,
		Name:     container.Metadata.Name,
		Platform: container.Spec.Platform,
		Image:    container.Spec.ImageName,
		HostID:   container.Spec.HostID,
		Agents:   strconv.Itoa(len(container.Status.AgentInstanceUIDs)),
	}
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return opt.Get(cmd, args)
	}

	return opt.List(cmd)
}

// List retrieves the list of containers.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	resp, err := opt.client.ContainerService.ListContainers(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	return opt.format(cmd, resp.Items)
}

// Get retrieves container(s) by ID.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	containers := make([]v1.Container, 0, len(ids))

	for _, id := range ids {
		result, err := opt.client.ContainerService.GetContainer(cmd.Context(), id)
		if err != nil {
			cmd.PrintErrf("failed to get container %s: %v\n", id, err)

			continue
		}

		containers = append(containers, *result)
	}

	if len(containers) == 0 {
		cmd.Println("No containers found.")

		return nil
	}

	return opt.format(cmd, containers)
}

func (opt *CommandOptions) format(cmd *cobra.Command, containers []v1.Container) error {
	formatType := formatter.FormatType(opt.formatType)

	var err error

	switch formatType {
	case formatter.SHORT, formatter.TEXT:
		items := lo.Map(containers, func(item v1.Container, _ int) ItemForCLI {
			return toItem(item)
		})
		err = formatter.Format(cmd.OutOrStdout(), items, formatType)
	case formatter.JSON, formatter.YAML:
		err = formatter.Format(cmd.OutOrStdout(), containers, formatType)
	default:
		return fmt.Errorf("unsupported format type: %s, %w", opt.formatType, ErrCommandExecutionFailed)
	}

	if err != nil {
		return fmt.Errorf("failed to format containers: %w", err)
	}

	return nil
}
