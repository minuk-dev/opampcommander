// Package whoami provides the whoami command for opampctl.
package whoami

import (
	"fmt"
	"time"

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

	email := "N/A"
	if info.Email != nil {
		email = *info.Email
	}

	//exhaustruct:ignore
	data := shortItemForCLI{
		Name:          currentUser.Name,
		AuthType:      currentUser.Auth.Type,
		Email:         email,
		Authenticated: info.Authenticated,
	}

	if o.outputFormat == "json" || o.outputFormat == "yaml" {
		detailErr := o.populateDetailedFields(cmd, &data)
		if detailErr != nil {
			return detailErr
		}
	}

	var formatType formatter.FormatType

	switch o.outputFormat {
	case "json":
		formatType = formatter.JSON
	case "yaml":
		formatType = formatter.YAML
	default:
		formatType = formatter.TEXT
	}

	err = formatter.Format(cmd.OutOrStdout(), data, formatType)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

func (o *CommandOptions) populateDetailedFields(cmd *cobra.Command, data *shortItemForCLI) error {
	profile, err := o.client.UserService.GetMyProfile(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get user profile: %w", err)
	}

	data.Labels = profile.User.Metadata.Labels

	roles, err := o.client.UserService.GetMyRoles(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get roles: %w", err)
	}

	bindings, err := o.client.UserService.GetMyRoleBindings(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get role bindings: %w", err)
	}

	bindingByRoleName := make(map[string]v1.RoleBinding, len(bindings.Items))

	for _, binding := range bindings.Items {
		if _, exists := bindingByRoleName[binding.Spec.RoleRef.Name]; !exists {
			bindingByRoleName[binding.Spec.RoleRef.Name] = binding
		}
	}

	data.Roles = make([]roleForCLI, 0, len(roles.Items))

	for _, role := range roles.Items {
		//exhaustruct:ignore
		roleCLI := roleForCLI{
			Name:        role.Spec.DisplayName,
			Description: role.Spec.Description,
			IsBuiltIn:   role.Spec.IsBuiltIn,
			Permissions: len(role.Spec.Permissions),
		}

		if binding, ok := bindingByRoleName[role.Spec.DisplayName]; ok {
			roleCLI.BindingReason = &roleBindingForCLI{
				Namespace: binding.Metadata.Namespace,
				Name:      binding.Metadata.Name,
				RoleRef: roleRefForCLI{
					Kind: binding.Spec.RoleRef.Kind,
					Name: binding.Spec.RoleRef.Name,
				},
				LabelSelector: binding.Spec.LabelSelector,
				CreatedAt:     binding.Metadata.CreatedAt.Time,
			}
		}

		data.Roles = append(data.Roles, roleCLI)
	}

	return nil
}

// shortItemForCLI is the top-level output structure for whoami.
// text/short format shows only the basic fields; json/yaml shows all fields.
type shortItemForCLI struct {
	Name          string            `json:"name"             text:"NAME"`
	AuthType      string            `json:"authType"         text:"AUTH_TYPE"`
	Email         string            `json:"email"            text:"EMAIL"`
	Authenticated bool              `json:"authenticated"    text:"AUTHENTICATED"`
	Labels        map[string]string `json:"labels,omitempty"`
	Roles         []roleForCLI      `json:"roles,omitempty"`
}

type roleForCLI struct {
	Name          string             `json:"name"`
	Description   string             `json:"description"`
	IsBuiltIn     bool               `json:"isBuiltIn"`
	Permissions   int                `json:"permissions"`
	BindingReason *roleBindingForCLI `json:"bindingReason,omitempty"`
}

type roleBindingForCLI struct {
	Namespace     string            `json:"namespace"`
	Name          string            `json:"name"`
	RoleRef       roleRefForCLI     `json:"roleRef"`
	LabelSelector map[string]string `json:"labelSelector,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
}

type roleRefForCLI struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}
