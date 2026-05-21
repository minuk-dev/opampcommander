// Package role provides the create role command for opampctl.
package role

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/create/internal/yamlfile"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// ErrDisplayNameRequired is returned when neither --display-name nor --file is given.
var ErrDisplayNameRequired = errors.New("--display-name is required (or use --file)")

// CommandOptions contains the options for the create role command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	displayName string
	description string
	permissions []string
	formatType  string
	file        string

	// internal state
	client *client.Client
}

// NewCommand creates a new create role command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Create a new role",
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

	cmd.Flags().StringVar(&options.displayName, "display-name", "",
		"Display name of the role (required unless --file is used)")
	cmd.Flags().StringVar(&options.description, "description", "", "Description of the role")
	cmd.Flags().StringSliceVar(&options.permissions, "permissions", nil, "Permission IDs for the role")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "text", "Output format (text, json, yaml)")
	cmd.Flags().StringVarP(&options.file, "file", "f", "",
		"Path to a full Role YAML definition. When set, individual CLI flags are ignored.")

	return cmd
}

// Prepare prepares the create role command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	cli, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = cli

	return nil
}

// Run executes the create role command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	createRequest, err := opt.buildRequest()
	if err != nil {
		return err
	}

	role, err := opt.client.RoleService.CreateRole(cmd.Context(), createRequest)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	type formattedRole struct {
		UID         string `json:"uid"         short:"UID"          text:"UID"          yaml:"uid"`
		DisplayName string `json:"displayName" short:"DISPLAY_NAME" text:"DISPLAY_NAME" yaml:"displayName"`
		Description string `json:"description" short:"DESCRIPTION"  text:"DESCRIPTION"  yaml:"description"`
	}

	formatted := &formattedRole{
		UID:         role.Metadata.UID,
		DisplayName: role.Spec.DisplayName,
		Description: role.Spec.Description,
	}

	err = formatter.Format(cmd.OutOrStdout(), formatted, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

func (opt *CommandOptions) buildRequest() (*v1.Role, error) {
	if opt.file != "" {
		//exhaustruct:ignore
		req := &v1.Role{}

		err := yamlfile.Load(opt.file, req)
		if err != nil {
			return nil, fmt.Errorf("load role from %s: %w", opt.file, err)
		}

		if req.Kind == "" {
			req.Kind = v1.RoleKind
		}

		if req.APIVersion == "" {
			req.APIVersion = v1.APIVersion
		}

		return req, nil
	}

	if opt.displayName == "" {
		return nil, ErrDisplayNameRequired
	}

	//exhaustruct:ignore
	return &v1.Role{
		Kind:       v1.RoleKind,
		APIVersion: v1.APIVersion,
		Spec: v1.RoleSpec{
			DisplayName: opt.displayName,
			Description: opt.description,
			Permissions: opt.permissions,
			IsBuiltIn:   false,
		},
	}, nil
}
