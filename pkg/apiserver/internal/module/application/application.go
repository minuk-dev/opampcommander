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
	certificateApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/certificate"
	containerApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/container"
	hostApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/host"
	namespaceApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/namespace"
	opampApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/opamp"
	roleApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/role"
	rolebindingApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/rolebinding"
	userApplicationService "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/user"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/helper"
)

// New creates a new module for application services.
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

			agentApplicationService.New,
			fx.Annotate(Identity[*agentApplicationService.Service], fx.As(new(port.AgentManageUsecase))),

			agentgroupApplicationService.NewManageService,
			fx.Annotate(Identity[*agentgroupApplicationService.ManageService], fx.As(new(port.AgentGroupManageUsecase))),

			agentpackageApplicationService.NewAgentPackageService,
			fx.Annotate(Identity[*agentpackageApplicationService.Service], fx.As(new(port.AgentPackageManageUsecase))),

			provideNamespaceService,
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

			// user & RBAC application services
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

// provideNamespaceService builds the namespace manage service, sourcing the
// undeletable default namespace name from configuration.
func provideNamespaceService(
	namespaceUsecase agentport.NamespaceUsecase,
	agentGroupUsecase agentport.AgentGroupUsecase,
	certificateUsecase agentport.CertificateUsecase,
	agentPackageUsecase agentport.AgentPackageUsecase,
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase,
	txRunner port.TransactionRunner,
	logger *slog.Logger,
	settings *config.ServerSettings,
) *namespaceApplicationService.Service {
	return namespaceApplicationService.NewNamespaceService(
		namespaceUsecase,
		agentGroupUsecase,
		certificateUsecase,
		agentPackageUsecase,
		agentRemoteConfigUsecase,
		txRunner,
		logger,
		settings.BootstrapSettings.DefaultNamespace,
	)
}

// Identity is a generic function that returns the input value.
// It is a helper function to generate a function that returns the input value.
// It is used to provide a function as a interface.
func Identity[T any](a T) T {
	return a
}
