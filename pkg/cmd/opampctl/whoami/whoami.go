// Package whoami provides the whoami command for opampctl.
package whoami

import (
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/configutil"
)

// CommandOptions contains the options for the whoami command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType       string
	showRoles        bool
	showRoleBindings bool

	// internal fields to run the command
	client *client.Client
}

// NewCommand creates a new whoami command.
func NewCommand(options CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Display the current user and context information with server validation",
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

	cmd.Flags().StringVarP(&options.formatType, "format", "f", "text", "Output format (text, json, yaml)")
	cmd.Flags().BoolVar(&options.showRoles, "roles", false, "Show roles assigned to the current user")
	cmd.Flags().BoolVar(&options.showRoleBindings, "rolebindings", false, "Show role bindings matching the current user (for debugging)")

	return cmd
}

// Prepare prepares the command options before running the command.
func (o *CommandOptions) Prepare(cmd *cobra.Command, _ []string) error {
	cfg, err := configutil.ApplyCmdFlags(o.GlobalConfig, cmd)
	if err != nil {
		return fmt.Errorf("failed to apply command flags: %w", err)
	}

	c, err := clientutil.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	o.client = c

	return nil
}

// Run executes display of the current user and context information.
func (o *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	currentUser := configutil.GetCurrentUser(o.GlobalConfig)

	info, err := o.client.AuthService.GetInfo()
	if err != nil {
		return fmt.Errorf("failed to get info from server: %w", err)
	}

	data := shortItemForCLI{
		Name:          currentUser.Name,
		AuthType:      currentUser.Auth.Type,
		Email:         switchIfNil(info.Email, "N/A"),
		Authenticated: info.Authenticated,
	}

	err = formatter.FormatText(cmd.OutOrStdout(), data)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	if o.showRoles {
		if printErr := o.printRoles(cmd); printErr != nil {
			return printErr
		}
	}

	if o.showRoleBindings {
		if printErr := o.printRoleBindings(cmd); printErr != nil {
			return printErr
		}
	}

	return nil
}

func (o *CommandOptions) printRoles(cmd *cobra.Command) error {
	resp, err := o.client.UserService.GetMyRoles(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get roles: %w", err)
	}

	if len(resp.Items) == 0 {
		cmd.Println("\nRoles: (none)")

		return nil
	}

	cmd.Println("\nRoles:")

	items := lo.Map(resp.Items, func(role v1.Role, _ int) formattedRole {
		return formattedRole{
			Name:        role.Spec.DisplayName,
			Description: role.Spec.Description,
			IsBuiltIn:   role.Spec.IsBuiltIn,
			Permissions: len(role.Spec.Permissions),
		}
	})

	if err := formatter.Format(cmd.OutOrStdout(), items, formatter.FormatType(o.formatType)); err != nil {
		return fmt.Errorf("failed to format roles: %w", err)
	}

	return nil
}

func (o *CommandOptions) printRoleBindings(cmd *cobra.Command) error {
	resp, err := o.client.UserService.GetMyRoleBindings(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get role bindings: %w", err)
	}

	if len(resp.Items) == 0 {
		cmd.Println("\nRoleBindings: (none)")

		return nil
	}

	cmd.Println("\nRoleBindings:")

	items := lo.Map(resp.Items, func(rb v1.RoleBinding, _ int) formattedRoleBinding {
		return toFormatted(rb)
	})

	if err := formatter.Format(cmd.OutOrStdout(), items, formatter.FormatType(o.formatType)); err != nil {
		return fmt.Errorf("failed to format role bindings: %w", err)
	}

	return nil
}

type shortItemForCLI struct {
	Name          string `json:"name"          short:"NAME"      text:"NAME"          yaml:"name"`
	AuthType      string `json:"authType"      short:"AUTH_TYPE" text:"AUTH_TYPE"     yaml:"authType"`
	Email         string `json:"email"         short:"EMAIL"     text:"EMAIL"         yaml:"email"`
	Authenticated bool   `json:"authenticated" short:"AUTH"      text:"AUTHENTICATED" yaml:"authenticated"`
}

type formattedRole struct {
	Name        string `json:"name"        short:"NAME"        text:"NAME"        yaml:"name"`
	Description string `json:"description" short:"DESCRIPTION" text:"DESCRIPTION" yaml:"description"`
	IsBuiltIn   bool   `json:"isBuiltIn"   short:"BUILT_IN"    text:"BUILT_IN"    yaml:"isBuiltIn"`
	Permissions int    `json:"permissions" short:"PERMISSIONS" text:"PERMISSIONS" yaml:"permissions"`
}

//nolint:lll
type formattedRoleBinding struct {
	Namespace     string            `json:"namespace"               short:"NAMESPACE"     text:"NAMESPACE"      yaml:"namespace"`
	Name          string            `json:"name"                    short:"NAME"          text:"NAME"           yaml:"name"`
	RoleRef       string            `json:"roleRef"                 short:"ROLE_REF"      text:"ROLE_REF"       yaml:"roleRef"`
	LabelSelector map[string]string `json:"labelSelector,omitempty" short:"LABEL_SEL"     text:"LABEL_SELECTOR" yaml:"labelSelector,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"               short:"CREATED_AT"    text:"CREATED_AT"     yaml:"createdAt"`
}

func toFormatted(rb v1.RoleBinding) formattedRoleBinding {
	return formattedRoleBinding{
		Namespace:     rb.Metadata.Namespace,
		Name:          rb.Metadata.Name,
		RoleRef:       rb.Spec.RoleRef.Kind + "/" + rb.Spec.RoleRef.Name,
		LabelSelector: rb.Spec.LabelSelector,
		CreatedAt:     rb.Metadata.CreatedAt.Time,
	}
}

func switchIfNil[T any](value *T, defaultValue T) T {
	if value == nil {
		return defaultValue
	}

	return *value
}
