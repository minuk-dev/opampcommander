// Package application provides the command to get application information.
package application

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/internal/selectorflags"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// ErrCommandExecutionFailed is returned when the command execution fails.
var ErrCommandExecutionFailed = errors.New("command execution failed")

// CommandOptions contains the options for the application command.
type CommandOptions struct {
	*config.GlobalConfig

	formatType string
	agents     bool
	selectors  selectorflags.Flags

	client *client.Client
}

// NewCommand creates a new application command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "application",
		Short: "get discovered applications",
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
	cmd.Flags().BoolVar(&options.agents, "agents", false, "List agents for the application ID")

	options.selectors.Register(cmd, selectorflags.Labels)

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

// ItemForCLI is a struct for application display.
type ItemForCLI struct {
	ID        string `short:"ID"        text:"ID"`
	Name      string `short:"Name"      text:"Name"`
	Namespace string `short:"Namespace" text:"Namespace"`
	Versions  string `short:"Versions"  text:"Versions"`
	Agents    string `short:"Agents"    text:"Agents"`
}

func toItem(application v1.Application) ItemForCLI {
	return ItemForCLI{
		ID:        application.Metadata.ID,
		Name:      application.Metadata.Name,
		Namespace: application.Spec.Namespace,
		Versions:  strings.Join(application.Spec.Versions, ","),
		Agents:    strconv.Itoa(len(application.Status.AgentInstanceUIDs)),
	}
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	if opt.agents {
		if len(args) != 1 {
			return fmt.Errorf("--agents requires exactly one application ID: %w", ErrCommandExecutionFailed)
		}

		return opt.listAgents(cmd, args[0])
	}

	if len(args) > 0 {
		return opt.Get(cmd, args)
	}

	return opt.List(cmd)
}

//nolint:funcorder // Kept beside Run because it implements its --agents branch.
func (opt *CommandOptions) listAgents(cmd *cobra.Command, id string) error {
	resp, err := opt.client.ApplicationService.ListAgentsByApplication(cmd.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to list application agents: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), resp.Items, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format application agents: %w", err)
	}

	return nil
}

// List retrieves the list of applications.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	listOpts, err := opt.selectors.ListOptions()
	if err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	resp, err := opt.client.ApplicationService.ListApplications(cmd.Context(), listOpts...)
	if err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	return opt.format(cmd, resp.Items)
}

// Get retrieves application(s) by ID.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	applications := make([]v1.Application, 0, len(ids))

	for _, id := range ids {
		result, err := opt.client.ApplicationService.GetApplication(cmd.Context(), id)
		if err != nil {
			cmd.PrintErrf("failed to get application %s: %v\n", id, err)

			continue
		}

		applications = append(applications, *result)
	}

	if len(applications) == 0 {
		cmd.Println("No applications found.")

		return nil
	}

	return opt.format(cmd, applications)
}

func (opt *CommandOptions) format(cmd *cobra.Command, applications []v1.Application) error {
	formatType := formatter.FormatType(opt.formatType)

	var err error

	switch formatType {
	case formatter.SHORT, formatter.TEXT:
		items := lo.Map(applications, func(item v1.Application, _ int) ItemForCLI {
			return toItem(item)
		})
		err = formatter.Format(cmd.OutOrStdout(), items, formatType)
	case formatter.JSON, formatter.YAML:
		err = formatter.Format(cmd.OutOrStdout(), applications, formatType)
	default:
		return fmt.Errorf("unsupported format type: %s, %w", opt.formatType, ErrCommandExecutionFailed)
	}

	if err != nil {
		return fmt.Errorf("failed to format applications: %w", err)
	}

	return nil
}
