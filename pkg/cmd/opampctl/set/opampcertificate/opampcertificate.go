// Package opampcertificate sets an agent group's OpAMP client certificate.
package opampcertificate

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/set/internal/resource"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

const maxGroupMembers = 2

var (
	errOneAgentRequired      = errors.New("rotation requires a group with exactly one agent")
	errOpAMPSettingsRequired = errors.New("group needs an OpAMP endpoint before certificate rotation")
	errCertificateCNMismatch = errors.New("certificate CN must equal the agent instance UID")
)

// CommandOptions contains flags for setting an agent group's OpAMP certificate.
type CommandOptions struct {
	*config.GlobalConfig

	namespace  string
	formatType string
}

// NewCommand creates the set opamp-certificate command.
func NewCommand(options CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opamp-certificate (agentgroup/NAME | agentgroup NAME) CERTIFICATE",
		Short: "Set the OpAMP client certificate for a single-agent group",
		Long: "Offer an existing Certificate to the group's agent. " +
			"Its leaf certificate CN must equal the agent instance UID.",
		Example: `  opampctl set opamp-certificate agentgroup/my-group new-cert
  opampctl set opamp-certificate agentgroup my-group new-cert -n default`,
		Args: cobra.RangeArgs(resource.MinArgs, resource.MaxArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, certificate, err := resource.Parse(args, "agentgroup")
			if err != nil {
				return fmt.Errorf("parse agent group target: %w", err)
			}

			return options.Run(cmd, name, certificate)
		},
	}
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace of the agent group")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "yaml", "Output format (yaml, json)")

	return cmd
}

// Run validates the certificate and updates the group's OpAMP offer.
func (opts *CommandOptions) Run(cmd *cobra.Command, name, certificateName string) error {
	cli, err := clientutil.NewClient(opts.GlobalConfig)
	if err != nil {
		return fmt.Errorf("create authenticated client: %w", err)
	}

	group, err := cli.AgentGroupService.GetAgentGroup(cmd.Context(), opts.namespace, name)
	if err != nil {
		return fmt.Errorf("get agent group: %w", err)
	}

	if group.Spec.AgentConfig == nil || group.Spec.AgentConfig.ConnectionSettings == nil ||
		group.Spec.AgentConfig.ConnectionSettings.OpAMP.DestinationEndpoint == "" {
		return errOpAMPSettingsRequired
	}

	members, err := cli.AgentGroupService.ListAgentsByAgentGroup(
		cmd.Context(), opts.namespace, name, client.WithLimit(maxGroupMembers),
	)
	if err != nil {
		return fmt.Errorf("list group agents: %w", err)
	}

	if len(members.Items) != 1 || members.Metadata.Continue != "" {
		return errOneAgentRequired
	}

	certificate, err := cli.CertificateService.GetCertificate(cmd.Context(), opts.namespace, certificateName)
	if err != nil {
		return fmt.Errorf("get certificate: %w", err)
	}

	uid := members.Items[0].Metadata.InstanceUID.String()

	err = validateCertificateForAgent(certificate, uid)
	if err != nil {
		return err
	}

	group.Spec.AgentConfig.ConnectionSettings.OpAMP.CertificateName = &certificateName

	updated, err := cli.AgentGroupService.UpdateAgentGroup(cmd.Context(), group)
	if err != nil {
		return fmt.Errorf("update agent group: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), updated, formatter.FormatType(opts.formatType))
	if err != nil {
		return fmt.Errorf("format agent group: %w", err)
	}

	return nil
}

func validateCertificateForAgent(certificate *v1.Certificate, uid string) error {
	keyPair, err := tls.X509KeyPair([]byte(certificate.Spec.Cert), []byte(certificate.Spec.PrivateKey))
	if err != nil {
		return fmt.Errorf("certificate resource has an invalid keypair: %w", err)
	}

	leaf, err := x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return fmt.Errorf("parse certificate leaf: %w", err)
	}

	if leaf.Subject.CommonName != uid {
		return fmt.Errorf("%w: got %q, want %s", errCertificateCNMismatch, leaf.Subject.CommonName, uid)
	}

	return nil
}
