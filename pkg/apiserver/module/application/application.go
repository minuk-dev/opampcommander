// Package application provides the application services module for the API server.
package application

import (
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/application/port"
	adminApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/admin"
	agentApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/agent"
	agentgroupApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/agentgroup"
	agentpackageApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/agentpackage"
	opampApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/opamp"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
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
		),
	)
}

// Identity is a generic function that returns the input value.
// It is a helper function to generate a function that returns the input value.
// It is used to provide a function as a interface.
func Identity[T any](a T) T {
	return a
}
