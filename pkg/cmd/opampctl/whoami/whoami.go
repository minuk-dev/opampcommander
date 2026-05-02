// Package whoami provides the whoami command for opampctl.
package whoami

import (
	"fmt"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

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
	outputFormat string

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

	cmd.Flags().StringVarP(&options.outputFormat, "output", "o", "text", "Output format (text, json, yaml)")

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
		Email:         mo.PointerToOption(info.Email).OrElse("N/A"),
		Authenticated: info.Authenticated,
		Labels:        nil,
		Roles:         nil,
	}

	err = formatter.FormatText(cmd.OutOrStdout(), data)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

type shortItemForCLI struct {
	Name          string            `json:"name"          short:"NAME"      text:"NAME"          yaml:"name"`
	AuthType      string            `json:"authType"      short:"AUTH_TYPE" text:"AUTH_TYPE"     yaml:"authType"`
	Email         string            `json:"email"         short:"EMAIL"     text:"EMAIL"         yaml:"email"`
	Authenticated bool              `json:"authenticated" short:"AUTH"      text:"AUTHENTICATED" yaml:"authenticated"`
	Labels        map[string]string `json:"labels" short:"LABELS" text:"LABELS" yaml:"labels"`

	Roles []roleForCLI `json:"roles", short:"ROLES", text:"ROLES" yaml:"roles"`
}

type roleForCLI struct {
	Name          string            `json:"name"        short:"NAME"        text:"NAME"        yaml:"name"`
	Description   string            `json:"description" short:"DESCRIPTION" text:"DESCRIPTION" yaml:"description"`
	IsBuiltIn     bool              `json:"isBuiltIn"   short:"BUILT_IN"    text:"BUILT_IN"    yaml:"isBuiltIn"`
	Permissions   int               `json:"permissions" short:"PERMISSIONS" text:"PERMISSIONS" yaml:"permissions"`
	BindingReason roleBindingForCLI `json:"bindingReason" short:"BINDING_REASON" "text:"BINDING_REASON" yaml:"bindingReason"`
}

//nolint:lll
type roleBindingForCLI struct {
	Namespace     string            `json:"namespace"               short:"NAMESPACE"  text:"NAMESPACE"      yaml:"namespace"`
	Name          string            `json:"name"                    short:"NAME"       text:"NAME"           yaml:"name"`
	RoleRef       roleRefForCLI     `json:"roleRef"                 short:"ROLE_REF"   text:"ROLE_REF"       yaml:"roleRef"`
	LabelSelector map[string]string `json:"labelSelector,omitempty" short:"LABEL_SEL"  text:"LABEL_SELECTOR" yaml:"labelSelector,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"               short:"CREATED_AT" text:"CREATED_AT"     yaml:"createdAt"`
}

type roleRefForCLI struct {
	Kind string `json:"kind" short:"KIND" text:"KIND" yaml:"kind"`
	Name string `json:"name" short:"NAME" text:"NAME" yaml:"name"`
}
