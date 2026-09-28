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
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/set/internal/resourceutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

const maxGroupMembers = 2

var (
	errOneAgentRequired      = errors.New("rotation requires a group with exactly one agent")
	errOpAMPSettingsRequired = errors.New("group needs an OpAMP endpoint before certificate rotation")
	errCertificateCNMismatch = errors.New("certificate CN must equal the agent instance UID")
)

// CommandOptions contains the options for setting an agent group's OpAMP certificate.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	namespace  string
	formatType string

	// internal
	client          *client.Client
	groupName       string
	certificateName string
	outputFormat    formatter.FormatType
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
		Args: cobra.RangeArgs(resourceutil.MinArgs, resourceutil.MaxArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := options.Prepare(cmd, args)
			if err != nil {
				return err
			}

			return options.Run(cmd, args)
		},
	}
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace of the agent group")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "yaml", "Output format (yaml, json)")

	return cmd
}

// Prepare parses the target and initializes the client.
func (opts *CommandOptions) Prepare(_ *cobra.Command, args []string) error {
	name, certificateName, err := resourceutil.Parse(args, "agentgroup")
	if err != nil {
		return fmt.Errorf("parse agent group target: %w", err)
	}

	cli, err := clientutil.NewClient(opts.GlobalConfig)
	if err != nil {
		return fmt.Errorf("create authenticated client: %w", err)
	}

	opts.groupName = name
	opts.certificateName = certificateName
	opts.client = cli
	opts.outputFormat = formatter.FormatType(opts.formatType)

	return nil
}

// Run validates the certificate and updates the group's OpAMP offer.
func (opts *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	group, err := opts.client.AgentGroupService.GetAgentGroup(cmd.Context(), opts.namespace, opts.groupName)
	if err != nil {
		return fmt.Errorf("get agent group: %w", err)
	}

	if group.Spec.AgentConfig == nil || group.Spec.AgentConfig.ConnectionSettings == nil ||
		group.Spec.AgentConfig.ConnectionSettings.OpAMP.DestinationEndpoint == "" {
		return errOpAMPSettingsRequired
	}

	members, err := opts.client.AgentGroupService.ListAgentsByAgentGroup(
		cmd.Context(), opts.namespace, opts.groupName, client.WithLimit(maxGroupMembers),
	)
	if err != nil {
		return fmt.Errorf("list group agents: %w", err)
	}

	if len(members.Items) != 1 || members.Metadata.Continue != "" {
		return errOneAgentRequired
	}

	certificate, err := opts.client.CertificateService.GetCertificate(
		cmd.Context(), opts.namespace, opts.certificateName,
	)
	if err != nil {
		return fmt.Errorf("get certificate: %w", err)
	}

	err = validateCertificateForAgent(certificate, members.Items[0].Metadata.InstanceUID.String())
	if err != nil {
		return err
	}

	group.Spec.AgentConfig.ConnectionSettings.OpAMP.CertificateName = &opts.certificateName

	updated, err := opts.client.AgentGroupService.UpdateAgentGroup(cmd.Context(), group)
	if err != nil {
		return fmt.Errorf("update agent group: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), updated, opts.outputFormat)
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
