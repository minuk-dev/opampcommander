// Package rolebinding implements the 'opampctl get rolebinding' command.
package rolebinding

import (
	"errors"
	"fmt"
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

// CommandOptions contains the options for the get rolebinding command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType string
	namespace  string

	// internal
	client *client.Client
}

// NewCommand creates a new get rolebinding command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "rolebinding",
		Short: "Get or list RoleBindings",
		Long: `Get or list RoleBindings in a namespace.

RoleBindings bind roles to users within namespaces, following the K8s RBAC pattern.

Examples:
  # List all rolebindings in default namespace
  opampctl get rolebinding

  # List rolebindings in production
  opampctl get rolebinding --namespace production

  # Get a specific rolebinding
  opampctl get rolebinding agent-viewer-production --namespace production`,
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
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace of the role binding")

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	cli, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = cli

	return nil
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return opt.list(cmd)
	}

	return opt.get(cmd, args)
}

func (opt *CommandOptions) list(cmd *cobra.Command) error {
	resp, err := opt.client.RoleBindingService.ListRoleBindings(cmd.Context(), opt.namespace)
	if err != nil {
		return fmt.Errorf("failed to list role bindings: %w", err)
	}

	items := lo.Map(resp.Items, func(rb v1.RoleBinding, _ int) formattedRoleBinding {
		return toFormatted(rb)
	})

	err = formatter.Format(cmd.OutOrStdout(), items, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

func (opt *CommandOptions) get(cmd *cobra.Command, names []string) error {
	type result struct {
		rb  *v1.RoleBinding
		err error
	}

	results := lo.Map(names, func(name string, _ int) result {
		rb, err := opt.client.RoleBindingService.GetRoleBinding(cmd.Context(), opt.namespace, name)

		return result{rb: rb, err: err}
	})

	items := lo.FilterMap(results, func(r result, _ int) (formattedRoleBinding, bool) {
		if r.err != nil {
			return formattedRoleBinding{}, false //nolint:exhaustruct
		}

		return toFormatted(*r.rb), true
	})

	if len(items) == 0 {
		cmd.Println("No role bindings found.")

		return nil
	}

	err := formatter.Format(cmd.OutOrStdout(), items, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

//nolint:lll
type formattedRoleBinding struct {
	Namespace string     `json:"namespace"           short:"namespace" text:"namespace"           yaml:"namespace"`
	Name      string     `json:"name"                short:"name"      text:"name"                yaml:"name"`
	RoleRef   string     `json:"roleRef"             short:"roleRef"   text:"roleRef"             yaml:"roleRef"`
	Subject   string     `json:"subject"             short:"subject"   text:"subject"             yaml:"subject"`
	CreatedAt time.Time  `json:"createdAt"           short:"createdAt" text:"createdAt"           yaml:"createdAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" short:"-"         text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
}

func toFormatted(roleBinding v1.RoleBinding) formattedRoleBinding {
	var deletedAt *time.Time

	if roleBinding.Metadata.DeletedAt != nil && !roleBinding.Metadata.DeletedAt.IsZero() {
		t := roleBinding.Metadata.DeletedAt.Time
		deletedAt = &t
	}

	return formattedRoleBinding{
		Namespace: roleBinding.Metadata.Namespace,
		Name:      roleBinding.Metadata.Name,
		RoleRef:   roleBinding.Spec.RoleRef.Kind + "/" + roleBinding.Spec.RoleRef.Name,
		Subject:   roleBinding.Spec.Subject.Kind + "/" + roleBinding.Spec.Subject.Name,
		CreatedAt: roleBinding.Metadata.CreatedAt.Time,
		DeletedAt: deletedAt,
	}
}
