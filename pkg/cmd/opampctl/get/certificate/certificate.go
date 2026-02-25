// Package certificate implements the 'opampctl get certificate' command.
package certificate

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

var (
	// ErrCommandExecutionFailed is returned when the command execution fails.
	ErrCommandExecutionFailed = errors.New("command execution failed")
)

// CommandOptions contains the options for the certificate command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType string

	// internal
	client *client.Client
}

// NewCommand creates a new certificate command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "certificate",
		Short: "Get certificate(s)",
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
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "short", "Output format (short, text, json, yaml)")

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	config := opt.GlobalConfig

	client, err := clientutil.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = client

	return nil
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		err := opt.List(cmd)
		if err != nil {
			return fmt.Errorf("list failed: %w", err)
		}

		return nil
	}

	names := args

	err := opt.Get(cmd, names)
	if err != nil {
		return fmt.Errorf("get failed: %w", err)
	}

	return nil
}

// List retrieves the list of certificates.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	certificates, err := clientutil.ListCertificateFully(cmd.Context(), opt.client)
	if err != nil {
		return fmt.Errorf("failed to list certificates: %w", err)
	}

	displayedCertificates := make([]formattedCertificate, len(certificates))
	for idx, cert := range certificates {
		displayedCertificates[idx] = opt.toFormattedCertificate(cert)
	}

	err = formatter.Format(cmd.OutOrStdout(), displayedCertificates, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format certificates: %w", err)
	}

	return nil
}

// Get retrieves the certificate information for the given names.
func (opt *CommandOptions) Get(cmd *cobra.Command, names []string) error {
	type CertificateWithErr struct {
		Certificate *v1.Certificate
		Err         error
	}

	certificatesWithErr := lo.Map(names, func(name string, _ int) CertificateWithErr {
		certificate, err := opt.client.CertificateService.GetCertificate(cmd.Context(), name)

		return CertificateWithErr{
			Certificate: certificate,
			Err:         err,
		}
	})

	certificates := lo.Filter(certificatesWithErr, func(c CertificateWithErr, _ int) bool {
		return c.Err == nil
	})
	if len(certificates) == 0 {
		cmd.Println("No certificates found or all specified certificates could not be retrieved.")

		return nil
	}

	displayedCertificates := lo.Map(certificates, func(c CertificateWithErr, _ int) formattedCertificate {
		return opt.toFormattedCertificate(*c.Certificate)
	})

	err := formatter.Format(cmd.OutOrStdout(), displayedCertificates, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format certificates: %w", err)
	}

	errs := lo.Filter(certificatesWithErr, func(c CertificateWithErr, _ int) bool {
		return c.Err != nil
	})
	if len(errs) > 0 {
		errMessages := lo.Map(errs, func(c CertificateWithErr, _ int) string {
			return c.Err.Error()
		})

		cmd.PrintErrf("Some certificates could not be retrieved: %s", strings.Join(errMessages, ", "))
	}

	return nil
}

//nolint:lll
type formattedCertificate struct {
	Name       string            `json:"name"                short:"name"      text:"name"                yaml:"name"`
	Attributes map[string]string `json:"attributes"          short:"-"         text:"-"                   yaml:"attributes"`
	HasCert    bool              `json:"hasCert"             short:"hasCert"   text:"hasCert"             yaml:"hasCert"`
	HasKey     bool              `json:"hasKey"              short:"hasKey"    text:"hasKey"              yaml:"hasKey"`
	HasCaCert  bool              `json:"hasCaCert"           short:"hasCaCert" text:"hasCaCert"           yaml:"hasCaCert"`
	CreatedAt  time.Time         `json:"createdAt"           short:"createdAt" text:"createdAt"           yaml:"createdAt"`
	CreatedBy  string            `json:"createdBy"           short:"createdBy" text:"createdBy"           yaml:"createdBy"`
	DeletedAt  *time.Time        `json:"deletedAt,omitempty" short:"-"         text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	DeletedBy  *string           `json:"deletedBy,omitempty" short:"-"         text:"deletedBy,omitempty" yaml:"deletedBy,omitempty"`
}

func extractConditionInfo(conditions []v1.Condition) (time.Time, string, *time.Time, *string) {
	var (
		createdAt time.Time
		createdBy string
		deletedAt *time.Time
		deletedBy *string
	)

	for _, condition := range conditions {
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

	return createdAt, createdBy, deletedAt, deletedBy
}

func (opt *CommandOptions) toFormattedCertificate(
	certificate v1.Certificate,
) formattedCertificate {
	// Extract timestamps from metadata first, then fallback to conditions
	var createdAt time.Time
	var createdBy string

	if certificate.Metadata.CreatedAt != nil && !certificate.Metadata.CreatedAt.IsZero() {
		createdAt = certificate.Metadata.CreatedAt.Time
	}

	// Get createdBy and deletedAt/deletedBy from conditions (createdBy is not in metadata)
	condCreatedAt, condCreatedBy, deletedAt, deletedBy := extractConditionInfo(certificate.Status.Conditions)
	createdBy = condCreatedBy

	// Fallback to condition's createdAt if metadata doesn't have it
	if createdAt.IsZero() {
		createdAt = condCreatedAt
	}

	return formattedCertificate{
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
