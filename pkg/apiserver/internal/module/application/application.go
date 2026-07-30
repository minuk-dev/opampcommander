// Package application provides the application services module for the API server.
package application

import (
	"log/slog"

	"go.uber.org/fx"

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
	remoteconfigschemaApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/remoteconfigschema"
	roleApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/role"
	rolebindingApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/rolebinding"
	serverApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/server"
	userApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/user"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/helper"
)

// customMessageHandlersGroup is the FX group into which OpAMP custom-message handlers are
// collected. A feature plugs a handler in by adding one AsCustomMessageHandler line below; the
// registry indexes them by capability and the ServerToAgentBuilder advertises those capabilities.
const customMessageHandlersGroup = `group:"opampCustomMessageHandlers"`

// New creates a new module for application services.
//
//nolint:funlen // DI wiring: a flat list of service providers/annotations.
func New() fx.Option {
	return fx.Module(
		"application",
		// application
		fx.Provide(
			opampApplicationService.New,
			fx.Annotate(Identity[*opampApplicationService.Service], fx.As(new(usecase.OpAMPUsecase))),
			helper.AsRunner(Identity[*opampApplicationService.Service]), // for background processing

			// Custom-message dispatch seam: the registry indexes the handlers collected in the
			// "opampCustomMessageHandlers" group (empty by default) and derives the custom
			// capabilities the ServerToAgentBuilder advertises. No handlers → no advertised
			// custom capabilities → behavior unchanged.
			fx.Annotate(
				opampApplicationService.NewCustomMessageRegistry,
				fx.ParamTags(customMessageHandlersGroup),
			),
			provideServerCustomCapabilities,

			adminApplicationService.New,
			fx.Annotate(Identity[*adminApplicationService.Service], fx.As(new(usecase.AdminUsecase))),
			serverApplicationService.New,
			fx.Annotate(Identity[*serverApplicationService.Service], fx.As(new(usecase.ServerManageUsecase))),

			agentApplicationService.New,
			fx.Annotate(Identity[*agentApplicationService.Service], fx.As(new(usecase.AgentManageUsecase))),

			reconcileApplicationService.New,
			fx.Annotate(Identity[*reconcileApplicationService.Service], fx.As(new(usecase.ReconcileManageUsecase))),

			agentgroupApplicationService.NewManageService,
			fx.Annotate(Identity[*agentgroupApplicationService.ManageService], fx.As(new(usecase.AgentGroupManageUsecase))),

			agentpackageApplicationService.NewAgentPackageService,
			fx.Annotate(Identity[*agentpackageApplicationService.Service], fx.As(new(usecase.AgentPackageManageUsecase))),

			namespaceApplicationService.NewNamespaceService,
			fx.Annotate(Identity[*namespaceApplicationService.Service], fx.As(new(usecase.NamespaceManageUsecase))),

			certificateApplicationService.NewCertificateService,
			fx.Annotate(Identity[*certificateApplicationService.Service], fx.As(new(usecase.CertificateManageUsecase))),

			hostApplicationService.New,
			fx.Annotate(Identity[*hostApplicationService.Service], fx.As(new(usecase.HostManageUsecase))),

			containerApplicationService.New,
			fx.Annotate(Identity[*containerApplicationService.Service], fx.As(new(usecase.ContainerManageUsecase))),

			agentremoteconfigApplicationService.NewAgentRemoteConfigService,
			fx.Annotate(
				Identity[*agentremoteconfigApplicationService.Service],
				fx.As(new(usecase.AgentRemoteConfigManageUsecase)),
			),

			endpointApplicationService.NewEndpointService,
			fx.Annotate(
				Identity[*endpointApplicationService.Service],
				fx.As(new(usecase.EndpointManageUsecase)),
			),

			remoteconfigschemaApplicationService.NewRemoteConfigSchemaService,
			fx.Annotate(
				Identity[*remoteconfigschemaApplicationService.Service],
				fx.As(new(usecase.RemoteConfigSchemaManageUsecase)),
			),

			provideEndpointMetricsService,
			fx.Annotate(
				Identity[*endpointmetricsApplicationService.Service],
				fx.As(new(usecase.EndpointMetricsUsecase)),
			),

			// user & RBAC application services
			authApplicationService.New,
			fx.Annotate(Identity[*authApplicationService.Service], fx.As(new(usecase.AuthProvisioningUsecase))),
			userApplicationService.New,
			fx.Annotate(Identity[*userApplicationService.Service], fx.As(new(usecase.UserManageUsecase))),

			roleApplicationService.New,
			fx.Annotate(Identity[*roleApplicationService.Service], fx.As(new(usecase.RoleManageUsecase))),

			rolebindingApplicationService.New,
			fx.Annotate(
				Identity[*rolebindingApplicationService.Service],
				fx.As(new(usecase.RoleBindingManageUsecase)),
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

// provideServerCustomCapabilities exposes the registry's advertised custom capabilities as the
// domain type the ServerToAgentBuilder consumes, keeping the registry the single source of truth
// for which custom capabilities the server offers.
func provideServerCustomCapabilities(
	registry *opampApplicationService.CustomMessageRegistry,
) agentservice.ServerCustomCapabilities {
	return registry.Capabilities()
}

// AsCustomMessageHandler annotates a CustomMessageHandler constructor so its result is added to
// the "opampCustomMessageHandlers" group consumed by the registry. This is the plug-in point for
// a feature that speaks a custom OpAMP capability.
func AsCustomMessageHandler(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(opampApplicationService.CustomMessageHandler)),
		fx.ResultTags(customMessageHandlersGroup),
	)
}

// Identity is a generic function that returns the input value.
// It is a helper function to generate a function that returns the input value.
// It is used to provide a function as a interface.
func Identity[T any](a T) T {
	return a
}
