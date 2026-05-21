// Package certificate provides the create certificate command for opampctl.
package certificate

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/create/internal/yamlfile"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// ErrNameRequired is returned when --name is missing and --file is not used.
var ErrNameRequired = errors.New("--name is required (or use --file)")

// CommandOptions contains the options for the create certificate command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	name       string
	namespace  string
	attributes map[string]string
	certFile   string
	keyFile    string
	caCertFile string
	cert       string
	key        string
	caCert     string
	file       string
	formatType string

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
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace of the certificate")
	cmd.Flags().StringToStringVar(&options.attributes, "attributes", nil, "Attributes of the certificate (key=value)")
	cmd.Flags().StringVar(&options.certFile, "cert-file", "", "Path to the certificate file (PEM)")
	cmd.Flags().StringVar(&options.keyFile, "key-file", "", "Path to the private key file (PEM)")
	cmd.Flags().StringVar(&options.caCertFile, "ca-cert-file", "", "Path to the CA certificate file (PEM)")
	cmd.Flags().StringVar(&options.cert, "cert", "", "Certificate content (PEM, alternative to --cert-file)")
	cmd.Flags().StringVar(&options.key, "key", "", "Private key content (PEM, alternative to --key-file)")
	cmd.Flags().StringVar(&options.caCert, "ca-cert", "", "CA certificate content (PEM, alternative to --ca-cert-file)")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "text", "Output format (text, json, yaml)")
	cmd.Flags().StringVarP(&options.file, "file", "f", "",
		"Path to a full Certificate YAML definition. When set, individual CLI flags are ignored.")

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
	createRequest, namespace, err := opt.buildRequest()
	if err != nil {
		return err
	}

	certificate, err := opt.client.CertificateService.CreateCertificate(cmd.Context(), namespace, createRequest)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), toFormattedCertificate(certificate), formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

func (opt *CommandOptions) buildRequest() (*v1.Certificate, string, error) {
	if opt.file != "" {
		//exhaustruct:ignore
		req := &v1.Certificate{}

		err := yamlfile.Load(opt.file, req)
		if err != nil {
			return nil, "", fmt.Errorf("load certificate from %s: %w", opt.file, err)
		}

		namespace := req.Metadata.Namespace
		if namespace == "" {
			namespace = opt.namespace
		}

		return req, namespace, nil
	}

	if opt.name == "" {
		return nil, "", ErrNameRequired
	}

	certContent, err := opt.loadCertContent()
	if err != nil {
		return nil, "", err
	}

	keyContent, err := opt.loadKeyContent()
	if err != nil {
		return nil, "", err
	}

	caCertContent, err := opt.loadCaCertContent()
	if err != nil {
		return nil, "", err
	}

	//exhaustruct:ignore
	return &v1.Certificate{
		Metadata: v1.CertificateMetadata{
			Name:       opt.name,
			Namespace:  opt.namespace,
			Attributes: opt.attributes,
		},
		Spec: v1.CertificateSpec{
			Cert:       certContent,
			PrivateKey: keyContent,
			CaCert:     caCertContent,
		},
	}, opt.namespace, nil
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
	DeletedAt  *time.Time        `json:"deletedAt,omitempty" short:"deletedAt,omitempty" text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
}

func toFormattedCertificate(certificate *v1.Certificate) *formattedCertificate {
	return &formattedCertificate{
		Name:       certificate.Metadata.Name,
		Attributes: certificate.Metadata.Attributes,
		HasCert:    certificate.Spec.Cert != "",
		HasKey:     certificate.Spec.PrivateKey != "",
		HasCaCert:  certificate.Spec.CaCert != "",
		DeletedAt:  switchToNilIfZero(certificate.Metadata.DeletedAt),
	}
}

func switchToNilIfZero(t *v1.Time) *time.Time {
	if t.IsZero() {
		return nil
	}

	return &t.Time
}
