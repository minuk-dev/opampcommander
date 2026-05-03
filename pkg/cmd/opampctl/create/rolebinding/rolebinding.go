// Package rolebinding provides the create rolebinding command for opampctl.
package rolebinding

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the create rolebinding command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	file       string
	formatType string

	// internal state
	client *client.Client
}

// NewCommand creates a new create rolebinding command.
//
//nolint:funlen // Long help text is intentional for CLI documentation.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "rolebinding",
		Short: "Create a RoleBinding from a YAML file",
		Long: `Create a RoleBinding from a YAML file.

A RoleBinding binds a Role to a User in a specific namespace,
granting the user all permissions defined in that Role within
that namespace scope.

YAML FORMAT:
  apiVersion: v1
  kind: RoleBinding
  metadata:
    name: <binding-name>
    namespace: <target-namespace>
  spec:
    roleRef:
      kind: Role
      name: <role-display-name>
    subjects:
      - kind: User
        name: <user-email>

EXAMPLES:

  1. Grant read-only access to agents in production:
     ---
     apiVersion: v1
     kind: RoleBinding
     metadata:
       name: agent-viewer-production
       namespace: production
     spec:
       roleRef:
         kind: Role
         name: Agent Viewer
       subjects:
         - kind: User
           name: alice@example.com

  2. Grant full CRUD on agents in staging:
     ---
     apiVersion: v1
     kind: RoleBinding
     metadata:
       name: agent-editor-staging
       namespace: staging
     spec:
       roleRef:
         kind: Role
         name: Agent Editor
       subjects:
         - kind: User
           name: bob@example.com

  3. Cluster-wide binding (like K8s ClusterRoleBinding):
     Grant agentgroup admin access across ALL namespaces:
     ---
     apiVersion: v1
     kind: RoleBinding
     metadata:
       name: cluster-agentgroup-admin
       namespace: "*"
     spec:
       roleRef:
         kind: Role
         name: AgentGroup Admin
       subjects:
         - kind: User
           name: ops-lead@example.com

  4. Grant certificate management in a specific namespace:
     ---
     apiVersion: v1
     kind: RoleBinding
     metadata:
       name: cert-manager-production
       namespace: production
     spec:
       roleRef:
         kind: Role
         name: Certificate Manager
       subjects:
         - kind: User
           name: security@example.com

  5. Multi-resource viewer (when the role includes multiple resources):
     ---
     apiVersion: v1
     kind: RoleBinding
     metadata:
       name: full-viewer-dev
       namespace: development
     spec:
       roleRef:
         kind: Role
         name: Full Viewer
       subjects:
         - kind: User
           name: junior-dev@example.com

  6. Agent package deployer in staging:
     ---
     apiVersion: v1
     kind: RoleBinding
     metadata:
       name: package-deployer-staging
       namespace: staging
     spec:
       roleRef:
         kind: Role
         name: Package Deployer
       subjects:
         - kind: User
           name: release-engineer@example.com

  7. Remote config editor for a team namespace:
     ---
     apiVersion: v1
     kind: RoleBinding
     metadata:
       name: config-editor-team-a
       namespace: team-a
     spec:
       roleRef:
         kind: Role
         name: Remote Config Editor
       subjects:
         - kind: User
           name: team-lead@example.com

USAGE:
  opampctl create rolebinding -f binding.yaml

COMMON ROLE EXAMPLES:
  Roles define which permissions are granted. Here are common role patterns:

  Agent Viewer      - agent:GET, agent:LIST
  Agent Editor      - agent:GET, agent:LIST, agent:CREATE, agent:UPDATE, agent:DELETE
  AgentGroup Admin  - agentgroup:GET, agentgroup:LIST, agentgroup:CREATE, agentgroup:UPDATE, agentgroup:DELETE
  Certificate Manager - certificate:GET, certificate:LIST, certificate:CREATE, certificate:UPDATE, certificate:DELETE
  Package Deployer  - agentpackage:GET, agentpackage:LIST, agentpackage:CREATE, agentpackage:UPDATE
  Remote Config Editor - agentremoteconfig:GET, agentremoteconfig:LIST,
    agentremoteconfig:CREATE, agentremoteconfig:UPDATE
  Full Viewer       - agent:GET, agent:LIST, agentgroup:GET, agentgroup:LIST, certificate:GET, certificate:LIST
  Super Editor      - All resources: GET, LIST, CREATE, UPDATE, DELETE`,
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

	cmd.Flags().StringVarP(&options.file, "file", "f", "", "Path to YAML file (required)")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "yaml", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("file") //nolint:errcheck,gosec

	return cmd
}

// Prepare prepares the create rolebinding command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	cli, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = cli

	return nil
}

// Run executes the create rolebinding command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	data, err := os.ReadFile(filepath.Clean(opt.file))
	if err != nil {
		return fmt.Errorf("failed to read YAML file: %w", err)
	}

	var roleBinding v1.RoleBinding

	err = yaml.Unmarshal(data, &roleBinding) //nolint:musttag // Condition type lacks yaml tags (pre-existing)
	if err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	created, err := opt.client.RoleBindingService.CreateRoleBinding(
		cmd.Context(), roleBinding.Metadata.Namespace, &roleBinding,
	)
	if err != nil {
		return fmt.Errorf("failed to create role binding: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), created, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}
