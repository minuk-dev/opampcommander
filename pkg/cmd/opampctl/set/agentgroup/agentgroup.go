// Package agentgroup sets connection settings on an agent group.
package agentgroup

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
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

	namespace   string
	certificate string
	formatType  string
}

// NewCommand creates the set agentgroup command.
func NewCommand(options CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agentgroup NAME",
		Short: "Rotate the OpAMP client certificate for a single-agent group",
		Long: "Offer an existing Certificate to the group's agent. " +
			"Its leaf certificate CN must equal the agent instance UID.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return options.Run(cmd, args[0])
		},
	}
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace of the agent group")
	cmd.Flags().StringVar(&options.certificate, "opamp-certificate", "", "Certificate resource to offer to the agent")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "yaml", "Output format (yaml, json)")
	_ = cmd.MarkFlagRequired("opamp-certificate")

	return cmd
}

// Run validates the certificate and updates the group's OpAMP offer.
func (opts *CommandOptions) Run(cmd *cobra.Command, name string) error {
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

	certificate, err := cli.CertificateService.GetCertificate(cmd.Context(), opts.namespace, opts.certificate)
	if err != nil {
		return fmt.Errorf("get certificate: %w", err)
	}

	uid := members.Items[0].Metadata.InstanceUID.String()

	err = validateCertificateForAgent(certificate, uid)
	if err != nil {
		return err
	}

	group.Spec.AgentConfig.ConnectionSettings.OpAMP.CertificateName = &opts.certificate

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
