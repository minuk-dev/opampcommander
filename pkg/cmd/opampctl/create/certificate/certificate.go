// Package certificate provides the create certificate command for opampctl.
package certificate

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the create certificate command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	name        string
	attributes  map[string]string
	certFile    string
	keyFile     string
	caCertFile  string
	cert        string
	key         string
	caCert      string
	formatType  string

	// internal state
	client *client.Client
}

// NewCommand creates a new create certificate command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "certificate",
		Short: "Create a certificate",
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

	cmd.Flags().StringVar(&options.name, "name", "", "Name of the certificate (required)")
	cmd.Flags().StringToStringVar(&options.attributes, "attributes", nil, "Attributes of the certificate (key=value)")
	cmd.Flags().StringVar(&options.certFile, "cert-file", "", "Path to the certificate file (PEM)")
	cmd.Flags().StringVar(&options.keyFile, "key-file", "", "Path to the private key file (PEM)")
	cmd.Flags().StringVar(&options.caCertFile, "ca-cert-file", "", "Path to the CA certificate file (PEM)")
	cmd.Flags().StringVar(&options.cert, "cert", "", "Certificate content (PEM, alternative to --cert-file)")
	cmd.Flags().StringVar(&options.key, "key", "", "Private key content (PEM, alternative to --key-file)")
	cmd.Flags().StringVar(&options.caCert, "ca-cert", "", "CA certificate content (PEM, alternative to --ca-cert-file)")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "text", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("name") //nolint:errcheck,gosec

	return cmd
}

// Prepare prepares the create certificate command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	client, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = client

	return nil
}

// Run executes the create certificate command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	certificateService := opt.client.CertificateService

	// Load certificate content from files if specified
	certContent, err := opt.loadCertContent()
	if err != nil {
		return err
	}

	keyContent, err := opt.loadKeyContent()
	if err != nil {
		return err
	}

	caCertContent, err := opt.loadCaCertContent()
	if err != nil {
		return err
	}

	//exhaustruct:ignore
	createRequest := &v1.Certificate{
		Metadata: v1.CertificateMetadata{
			Name:       opt.name,
			Attributes: opt.attributes,
		},
		Spec: v1.CertificateSpec{
			Cert:       certContent,
			PrivateKey: keyContent,
			CaCert:     caCertContent,
		},
	}

	certificate, err := certificateService.CreateCertificate(cmd.Context(), createRequest)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), toFormattedCertificate(certificate), formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

func (opt *CommandOptions) loadCertContent() (string, error) {
	if opt.certFile != "" {
		content, err := os.ReadFile(opt.certFile)
		if err != nil {
			return "", fmt.Errorf("failed to read certificate file: %w", err)
		}

		return string(content), nil
	}

	return opt.cert, nil
}

func (opt *CommandOptions) loadKeyContent() (string, error) {
	if opt.keyFile != "" {
		content, err := os.ReadFile(opt.keyFile)
		if err != nil {
			return "", fmt.Errorf("failed to read private key file: %w", err)
		}

		return string(content), nil
	}

	return opt.key, nil
}

func (opt *CommandOptions) loadCaCertContent() (string, error) {
	if opt.caCertFile != "" {
		content, err := os.ReadFile(opt.caCertFile)
		if err != nil {
			return "", fmt.Errorf("failed to read CA certificate file: %w", err)
		}

		return string(content), nil
	}

	return opt.caCert, nil
}

//nolint:lll
type formattedCertificate struct {
	Name       string            `json:"name"                short:"name"                text:"name"                yaml:"name"`
	Attributes map[string]string `json:"attributes"          short:"-"                   text:"-"                   yaml:"attributes"`
	HasCert    bool              `json:"hasCert"             short:"hasCert"             text:"hasCert"             yaml:"hasCert"`
	HasKey     bool              `json:"hasKey"              short:"hasKey"              text:"hasKey"              yaml:"hasKey"`
	HasCaCert  bool              `json:"hasCaCert"           short:"hasCaCert"           text:"hasCaCert"           yaml:"hasCaCert"`
	CreatedAt  time.Time         `json:"createdAt"           short:"createdAt"           text:"createdAt"           yaml:"createdAt"`
	CreatedBy  string            `json:"createdBy"           short:"createdBy"           text:"createdBy"           yaml:"createdBy"`
	DeletedAt  *time.Time        `json:"deletedAt,omitempty" short:"deletedAt,omitempty" text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	DeletedBy  *string           `json:"deletedBy,omitempty" short:"deletedBy,omitempty" text:"deletedBy,omitempty" yaml:"deletedBy,omitempty"`
}

func toFormattedCertificate(certificate *v1.Certificate) *formattedCertificate {
	// Extract timestamps and users from conditions
	var (
		createdAt time.Time
		createdBy string
		deletedAt *time.Time
		deletedBy *string
	)

	for _, condition := range certificate.Status.Conditions {
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

	return &formattedCertificate{
		Name:       certificate.Metadata.Name,
		Attributes: certificate.Metadata.Attributes,
		HasCert:    certificate.Spec.Cert != "",
		HasKey:     certificate.Spec.PrivateKey != "",
		HasCaCert:  certificate.Spec.CaCert != "",
		CreatedAt:  createdAt,
		CreatedBy:  createdBy,
		DeletedAt:  deletedAt,
		DeletedBy:  deletedBy,
	}
}
