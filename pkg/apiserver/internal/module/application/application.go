// Package application provides the application services module for the API server.
package application

import (
	"log/slog"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	adminApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/admin"
	agentApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agent"
	agentgroupApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agentgroup"
	agentpackageApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agentpackage"
	agentremoteconfigApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agentremoteconfig"
	authApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/auth"
	certificateApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/certificate"
	containerApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/container"
	endpointApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/endpoint"
	endpointmetricsApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/endpointmetrics"
	hostApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/host"
	namespaceApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/namespace"
	opampApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/opamp"
	reconcileApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/reconcile"
	roleApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/role"
	rolebindingApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/rolebinding"
	serverApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/server"
	userApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/user"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/helper"
)

// New creates a new module for application services.
//
//nolint:funlen // DI wiring: a flat list of service providers/annotations.
func New() fx.Option {
	return fx.Module(
		"application",
		// application
		fx.Provide(
			opampApplicationService.New,
			fx.Annotate(Identity[*opampApplicationService.Service], fx.As(new(port.OpAMPUsecase))),
			helper.AsRunner(Identity[*opampApplicationService.Service]), // for background processing

			adminApplicationService.New,
			fx.Annotate(Identity[*adminApplicationService.Service], fx.As(new(port.AdminUsecase))),
			serverApplicationService.New,
			fx.Annotate(Identity[*serverApplicationService.Service], fx.As(new(port.ServerManageUsecase))),

			agentApplicationService.New,
			fx.Annotate(Identity[*agentApplicationService.Service], fx.As(new(port.AgentManageUsecase))),

			reconcileApplicationService.New,
			fx.Annotate(Identity[*reconcileApplicationService.Service], fx.As(new(port.ReconcileManageUsecase))),

			agentgroupApplicationService.NewManageService,
			fx.Annotate(Identity[*agentgroupApplicationService.ManageService], fx.As(new(port.AgentGroupManageUsecase))),

			agentpackageApplicationService.NewAgentPackageService,
			fx.Annotate(Identity[*agentpackageApplicationService.Service], fx.As(new(port.AgentPackageManageUsecase))),

			namespaceApplicationService.NewNamespaceService,
			fx.Annotate(Identity[*namespaceApplicationService.Service], fx.As(new(port.NamespaceManageUsecase))),

			certificateApplicationService.NewCertificateService,
			fx.Annotate(Identity[*certificateApplicationService.Service], fx.As(new(port.CertificateManageUsecase))),

			hostApplicationService.New,
			fx.Annotate(Identity[*hostApplicationService.Service], fx.As(new(port.HostManageUsecase))),

			containerApplicationService.New,
			fx.Annotate(Identity[*containerApplicationService.Service], fx.As(new(port.ContainerManageUsecase))),

			agentremoteconfigApplicationService.NewAgentRemoteConfigService,
			fx.Annotate(
				Identity[*agentremoteconfigApplicationService.Service],
				fx.As(new(port.AgentRemoteConfigManageUsecase)),
			),

			endpointApplicationService.NewEndpointService,
			fx.Annotate(
				Identity[*endpointApplicationService.Service],
				fx.As(new(port.EndpointManageUsecase)),
			),

			provideEndpointMetricsService,
			fx.Annotate(
				Identity[*endpointmetricsApplicationService.Service],
				fx.As(new(port.EndpointMetricsUsecase)),
			),

			// user & RBAC application services
			authApplicationService.New,
			fx.Annotate(Identity[*authApplicationService.Service], fx.As(new(port.AuthProvisioningUsecase))),
			userApplicationService.New,
			fx.Annotate(Identity[*userApplicationService.Service], fx.As(new(port.UserManageUsecase))),

			roleApplicationService.New,
			fx.Annotate(Identity[*roleApplicationService.Service], fx.As(new(port.RoleManageUsecase))),

			rolebindingApplicationService.New,
			fx.Annotate(
				Identity[*rolebindingApplicationService.Service],
				fx.As(new(port.RoleBindingManageUsecase)),
			),
		),
	)
}

// provideEndpointMetricsService builds the endpoint-throughput service, sourcing
// the default rate window from configuration.
func provideEndpointMetricsService(
	usecase agentport.EndpointMetricsUsecase,
	logger *slog.Logger,
	settings *config.ServerSettings,
) *endpointmetricsApplicationService.Service {
	return endpointmetricsApplicationService.NewEndpointMetricsService(
		usecase,
		settings.MetricsBackend.DefaultWindow,
		logger,
	)
}

// Identity is a generic function that returns the input value.
// It is a helper function to generate a function that returns the input value.
// It is used to provide a function as a interface.
func Identity[T any](a T) T {
	return a
}
