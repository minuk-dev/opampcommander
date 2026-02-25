// Package agentpackage implements the 'opampctl get agentpackage' command.
package agentpackage

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

var (
	// ErrCommandExecutionFailed is returned when the command execution fails.
	ErrCommandExecutionFailed = errors.New("command execution failed")
)

// CommandOptions contains the options for the agentpackage command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType string

	// internal
	client *client.Client
}

// NewCommand creates a new agentpackage command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agentpackage",
		Short: "agentpackage",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := options.Prepare(cmd, args)
			if err != nil {
				return err
			}

			err = options.Run(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "short", "Output format (short, text, json, yaml)")

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	config := opt.GlobalConfig

	client, err := clientutil.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = client

	return nil
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		err := opt.List(cmd)
		if err != nil {
			return fmt.Errorf("list failed: %w", err)
		}

		return nil
	}

	names := args

	err := opt.Get(cmd, names)
	if err != nil {
		return fmt.Errorf("get failed: %w", err)
	}

	return nil
}

// List retrieves the list of agent packages.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	agentpackages, err := clientutil.ListAgentPackageFully(cmd.Context(), opt.client)
	if err != nil {
		return fmt.Errorf("failed to list agent packages: %w", err)
	}

	displayedPackages := make([]formattedAgentPackage, len(agentpackages))
	for idx, agentpackage := range agentpackages {
		displayedPackages[idx] = opt.toFormattedAgentPackage(agentpackage)
	}

	err = formatter.Format(cmd.OutOrStdout(), displayedPackages, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format agentpackage: %w", err)
	}

	return nil
}

// Get retrieves the agent package information for the given names.
func (opt *CommandOptions) Get(cmd *cobra.Command, names []string) error {
	type AgentPackageWithErr struct {
		AgentPackage *v1.AgentPackage
		Err          error
	}

	agentPackagesWithErr := lo.Map(names, func(name string, _ int) AgentPackageWithErr {
		agentPackage, err := opt.client.AgentPackageService.GetAgentPackage(cmd.Context(), name)

		return AgentPackageWithErr{
			AgentPackage: agentPackage,
			Err:          err,
		}
	})

	agentPackages := lo.Filter(agentPackagesWithErr, func(a AgentPackageWithErr, _ int) bool {
		return a.Err == nil
	})
	if len(agentPackages) == 0 {
		cmd.Println("No agent packages found or all specified packages could not be retrieved.")

		return nil
	}

	displayedAgentPackages := lo.Map(agentPackages, func(a AgentPackageWithErr, _ int) formattedAgentPackage {
		return opt.toFormattedAgentPackage(*a.AgentPackage)
	})

	err := formatter.Format(cmd.OutOrStdout(), displayedAgentPackages, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format agent packages: %w", err)
	}

	errs := lo.Filter(agentPackagesWithErr, func(a AgentPackageWithErr, _ int) bool {
		return a.Err != nil
	})
	if len(errs) > 0 {
		errMessages := lo.Map(errs, func(a AgentPackageWithErr, _ int) string {
			return a.Err.Error()
		})

		cmd.PrintErrf("Some agent packages could not be retrieved: %s", strings.Join(errMessages, ", "))
	}

	return nil
}

//nolint:lll
type formattedAgentPackage struct {
	Name        string            `json:"name"                short:"name"      text:"name"                yaml:"name"`
	Attributes  map[string]string `json:"attributes"          short:"-"         text:"-"                   yaml:"attributes"`
	PackageType string            `json:"packageType"         short:"type"      text:"packageType"         yaml:"packageType"`
	Version     string            `json:"version"             short:"version"   text:"version"             yaml:"version"`
	DownloadURL string            `json:"downloadUrl"         short:"-"         text:"downloadUrl"         yaml:"downloadUrl"`
	CreatedAt   *time.Time        `json:"createdAt,omitempty" short:"createdAt" text:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	CreatedBy   string            `json:"createdBy,omitempty" short:"createdBy" text:"createdBy,omitempty" yaml:"createdBy,omitempty"`
	DeletedAt   *time.Time        `json:"deletedAt,omitempty" short:"-"         text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
}

func extractConditionInfo(conditions []v1.Condition) (time.Time, string, *time.Time, *string) {
	var (
		createdAt time.Time
		createdBy string
		deletedAt *time.Time
		deletedBy *string
	)

	for _, condition := range conditions {
		switch condition.Type { //nolint:exhaustive // Only handle Created and Deleted conditions
		case v1.ConditionTypeCreated:
			if condition.Status == v1.ConditionStatusTrue {
				createdAt = condition.LastTransitionTime.Time
				createdBy = condition.Reason
			}
		case v1.ConditionTypeDeleted:
			if condition.Status == v1.ConditionStatusTrue {
				t := condition.LastTransitionTime.Time
				deletedAt = &t
				deletedBy = &condition.Reason
			}
		}
	}

	return createdAt, createdBy, deletedAt, deletedBy
}

func (opt *CommandOptions) toFormattedAgentPackage(
	agentPackage v1.AgentPackage,
) formattedAgentPackage {
	// Extract timestamps from metadata first, then fallback to conditions
	var (
		createdAt *time.Time
		createdBy string
	)

	if agentPackage.Metadata.CreatedAt != nil && !agentPackage.Metadata.CreatedAt.IsZero() {
		createdAt = &agentPackage.Metadata.CreatedAt.Time
	}

	// Get createdBy from conditions (createdBy is not in metadata)
	condCreatedAt, condCreatedBy, _, _ := extractConditionInfo(agentPackage.Status.Conditions)
	createdBy = condCreatedBy

	// Fallback to condition's createdAt if metadata doesn't have it
	if createdAt == nil && !condCreatedAt.IsZero() {
		createdAt = &condCreatedAt
	}

	return formattedAgentPackage{
		Name:        agentPackage.Metadata.Name,
		Attributes:  agentPackage.Metadata.Attributes,
		PackageType: agentPackage.Spec.PackageType,
		Version:     agentPackage.Spec.Version,
		DownloadURL: agentPackage.Spec.DownloadURL,
		CreatedAt:   createdAt,
		CreatedBy:   createdBy,
		DeletedAt:   switchToNilIfZero(agentPackage.Metadata.DeletedAt),
	}
}

func switchToNilIfZero(t *v1.Time) *time.Time {
	if t.IsZero() {
		return nil
	}

	return &t.Time
}
