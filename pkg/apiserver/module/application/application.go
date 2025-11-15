// Package application provides the application services module for the API server.
package application

import (
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/application/port"
	adminApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/admin"
	agentApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/agent"
	agentgroupApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/agentgroup"
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
			fx.Annotate(helper.Identity[*opampApplicationService.Service], fx.As(new(port.OpAMPUsecase))),
			helper.AsRunner(helper.Identity[*opampApplicationService.Service]), // for background processing

			adminApplicationService.New,
			fx.Annotate(helper.Identity[*adminApplicationService.Service], fx.As(new(port.AdminUsecase))),

			agentApplicationService.New,
			fx.Annotate(helper.Identity[*agentApplicationService.Service], fx.As(new(port.AgentManageUsecase))),

			agentgroupApplicationService.NewManageService,
			fx.Annotate(helper.Identity[*agentgroupApplicationService.ManageService], fx.As(new(port.AgentGroupManageUsecase))),
		),
	)
}
