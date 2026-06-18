// Package endpoint implements the 'opampctl get endpoint' command.
package endpoint

import (
	"context"
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

// CommandOptions contains the options for the endpoint command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType     string
	includeDeleted bool
	namespace      string
	allNamespaces  bool

	// internal
	client *client.Client
}

// NewCommand creates a new endpoint command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "endpoint",
		Short: "endpoint",
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
	cmd.Flags().BoolVar(&options.includeDeleted, "include-deleted", false, "Include soft-deleted endpoints")
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace")
	cmd.Flags().BoolVarP(&options.allNamespaces, "all-namespaces", "A", false, "List resources across all namespaces")

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

// List retrieves the list of endpoints.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	listOpts := []client.ListOption{client.WithIncludeDeleted(opt.includeDeleted)}

	var (
		endpoints []v1.Endpoint
		err       error
	)

	if opt.allNamespaces {
		endpoints, err = opt.listAllNamespaces(cmd, listOpts...)
	} else {
		endpoints, err = clientutil.ListEndpointFully(
			cmd.Context(), opt.client, opt.namespace, listOpts...,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to list endpoints: %w", err)
	}

	displayed := make([]formattedEndpoint, len(endpoints))
	for idx, endpoint := range endpoints {
		displayed[idx] = toFormattedEndpoint(endpoint)
	}

	err = formatter.Format(cmd.OutOrStdout(), displayed, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format endpoint: %w", err)
	}

	return nil
}

// Get retrieves the endpoint information for the given names.
func (opt *CommandOptions) Get(cmd *cobra.Command, names []string) error {
	type endpointWithErr struct {
		Endpoint *v1.Endpoint
		Err      error
	}

	getOpts := []client.GetOption{client.WithGetIncludeDeleted(opt.includeDeleted)}

	endpointsWithErr := lo.Map(names, func(name string, _ int) endpointWithErr {
		endpoint, err := opt.client.EndpointService.GetEndpoint(
			cmd.Context(), opt.namespace, name, getOpts...)

		return endpointWithErr{
			Endpoint: endpoint,
			Err:      err,
		}
	})

	endpoints := lo.Filter(endpointsWithErr, func(e endpointWithErr, _ int) bool {
		return e.Err == nil
	})
	if len(endpoints) == 0 {
		cmd.Println("No endpoints found or all specified endpoints could not be retrieved.")

		return nil
	}

	displayed := lo.Map(endpoints, func(e endpointWithErr, _ int) formattedEndpoint {
		return toFormattedEndpoint(*e.Endpoint)
	})

	err := formatter.Format(cmd.OutOrStdout(), displayed, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format endpoints: %w", err)
	}

	errs := lo.Filter(endpointsWithErr, func(e endpointWithErr, _ int) bool {
		return e.Err != nil
	})
	if len(errs) > 0 {
		errMessages := lo.Map(errs, func(e endpointWithErr, _ int) string {
			return e.Err.Error()
		})

		cmd.PrintErrf("Some endpoints could not be retrieved: %s", strings.Join(errMessages, ", "))
	}

	return nil
}

func (opt *CommandOptions) listAllNamespaces(
	cmd *cobra.Command, listOpts ...client.ListOption,
) ([]v1.Endpoint, error) {
	endpoints, err := clientutil.ListAcrossNamespaces(
		cmd.Context(), opt.client,
		func(ctx context.Context, namespace string) ([]v1.Endpoint, error) {
			return clientutil.ListEndpointFully(ctx, opt.client, namespace, listOpts...)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints across all namespaces: %w", err)
	}

	return endpoints, nil
}

//nolint:lll
type formattedEndpoint struct {
	Namespace  string            `json:"namespace"           short:"namespace" text:"namespace"           yaml:"namespace"`
	Name       string            `json:"name"                short:"name"      text:"name"                yaml:"name"`
	URL        string            `json:"url"                 short:"url"       text:"url"                 yaml:"url"`
	Protocol   string            `json:"protocol"            short:"protocol"  text:"protocol"            yaml:"protocol"`
	Signals    string            `json:"signals"             short:"signals"   text:"signals"             yaml:"signals"`
	Tenants    int               `json:"tenants"             short:"tenants"   text:"tenants"             yaml:"tenants"`
	Attributes map[string]string `json:"attributes"          short:"-"         text:"-"                   yaml:"attributes"`
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

// signalsString renders the supported signals as a compact, stable string such
// as "metrics,traces"; it returns "-" when no signal is supported.
func signalsString(signals v1.EndpointSignals) string {
	enabled := make([]string, 0)
	if signals.Metrics {
		enabled = append(enabled, "metrics")
	}

	if signals.Logs {
		enabled = append(enabled, "logs")
	}

	if signals.Traces {
		enabled = append(enabled, "traces")
	}

	if len(enabled) == 0 {
		return "-"
	}

	return strings.Join(enabled, ",")
}

func toFormattedEndpoint(endpoint v1.Endpoint) formattedEndpoint {
	createdAt := endpoint.Metadata.CreatedAt.Time

	condCreatedAt, createdBy, deletedAt, deletedBy := extractConditionInfo(endpoint.Status.Conditions)
	if createdAt.IsZero() {
		createdAt = condCreatedAt
	}

	return formattedEndpoint{
		Namespace:  endpoint.Metadata.Namespace,
		Name:       endpoint.Metadata.Name,
		URL:        endpoint.Spec.URL,
		Protocol:   endpoint.Spec.Protocol,
		Signals:    signalsString(endpoint.Spec.Signals),
		Tenants:    len(endpoint.Spec.Tenants),
		Attributes: endpoint.Metadata.Attributes,
		CreatedAt:  createdAt,
		CreatedBy:  createdBy,
		DeletedAt:  deletedAt,
		DeletedBy:  deletedBy,
	}
}
