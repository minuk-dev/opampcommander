// Package endpoint provides the create endpoint command for opampctl.
package endpoint

import (
	"errors"
	"fmt"
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

// CommandOptions contains the options for the create endpoint command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	name       string
	namespace  string
	attributes map[string]string
	url        string
	protocol   string
	metrics    bool
	logs       bool
	traces     bool
	file       string
	formatType string

	// internal state
	client *client.Client
}

// NewCommand creates a new create endpoint command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "endpoint",
		Short: "create endpoint",
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

	cmd.Flags().StringVar(&options.name, "name", "", "Name of the endpoint (required unless --file is used)")
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace")
	cmd.Flags().StringToStringVar(&options.attributes, "attributes", nil, "Attributes of the endpoint (key=value)")
	cmd.Flags().StringVar(&options.url, "url", "", "Destination URL of the telemetry backend")
	cmd.Flags().StringVar(&options.protocol, "protocol", "",
		"Export protocol (e.g. otlp, otlphttp, prometheusremotewrite)")
	cmd.Flags().BoolVar(&options.metrics, "metrics", false, "Endpoint supports the metrics signal")
	cmd.Flags().BoolVar(&options.logs, "logs", false, "Endpoint supports the logs signal")
	cmd.Flags().BoolVar(&options.traces, "traces", false, "Endpoint supports the traces signal")
	cmd.Flags().StringVarP(&options.file, "file", "f", "",
		"Path to an endpoint YAML/JSON definition (use for multi-tenant endpoints)")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "text", "Output format (text, json, yaml)")

	return cmd
}

// Prepare prepares the create endpoint command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	client, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = client

	return nil
}

// Run executes the create endpoint command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	createRequest, namespace, err := opt.buildRequest()
	if err != nil {
		return err
	}

	endpoint, err := opt.client.EndpointService.CreateEndpoint(cmd.Context(), namespace, createRequest)
	if err != nil {
		return fmt.Errorf("failed to create endpoint: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), toFormattedEndpoint(endpoint), formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

func (opt *CommandOptions) buildRequest() (*v1.Endpoint, string, error) {
	if opt.file != "" {
		//exhaustruct:ignore
		req := &v1.Endpoint{}

		err := yamlfile.Load(opt.file, req)
		if err != nil {
			return nil, "", fmt.Errorf("load endpoint from %s: %w", opt.file, err)
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

	//exhaustruct:ignore
	return &v1.Endpoint{
		Metadata: v1.EndpointMetadata{
			Name:       opt.name,
			Namespace:  opt.namespace,
			Attributes: opt.attributes,
		},
		Spec: v1.EndpointSpec{
			URL:      opt.url,
			Protocol: opt.protocol,
			Signals: v1.EndpointSignals{
				Metrics: opt.metrics,
				Logs:    opt.logs,
				Traces:  opt.traces,
			},
		},
	}, opt.namespace, nil
}

type formattedEndpoint struct {
	Name       string            `json:"name"       short:"name"      text:"name"      yaml:"name"`
	Attributes map[string]string `json:"attributes" short:"-"         text:"-"         yaml:"attributes"`
	URL        string            `json:"url"        short:"url"       text:"url"       yaml:"url"`
	Protocol   string            `json:"protocol"   short:"protocol"  text:"protocol"  yaml:"protocol"`
	CreatedAt  time.Time         `json:"createdAt"  short:"createdAt" text:"createdAt" yaml:"createdAt"`
	CreatedBy  string            `json:"createdBy"  short:"createdBy" text:"createdBy" yaml:"createdBy"`
}

func toFormattedEndpoint(endpoint *v1.Endpoint) *formattedEndpoint {
	var (
		createdAt time.Time
		createdBy string
	)

	for _, condition := range endpoint.Status.Conditions {
		if condition.Type == v1.ConditionTypeCreated && condition.Status == v1.ConditionStatusTrue {
			createdAt = condition.LastTransitionTime.Time
			createdBy = condition.Reason
		}
	}

	if createdAt.IsZero() {
		createdAt = endpoint.Metadata.CreatedAt.Time
	}

	return &formattedEndpoint{
		Name:       endpoint.Metadata.Name,
		Attributes: endpoint.Metadata.Attributes,
		URL:        endpoint.Spec.URL,
		Protocol:   endpoint.Spec.Protocol,
		CreatedAt:  createdAt,
		CreatedBy:  createdBy,
	}
}
