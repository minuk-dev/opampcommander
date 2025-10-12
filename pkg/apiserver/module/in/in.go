package in

import (
	"context"
	"net"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/auth/basic"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/auth/github"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/command"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/connection"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/opamp"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/ping"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/version"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
)

// New creates a new module for controllers.
func New() fx.Option {
	return fx.Module(
		"inport",
		// base
		fx.Provide(
			NewHTTPServer,
			fx.Annotate(NewEngine, fx.ParamTags(`group:"controllers"`)),
		),
		fx.Provide(
			ping.NewController, helper.AsController(helper.Identity[*ping.Controller]),
			opamp.NewController, helper.AsController(helper.Identity[*opamp.Controller]),
			version.NewController, helper.AsController(helper.Identity[*version.Controller]),
			connection.NewController, helper.AsController(helper.Identity[*connection.Controller]),
			agent.NewController, helper.AsController(helper.Identity[*agent.Controller]),
			agentgroup.NewController, helper.AsController(helper.Identity[*agentgroup.Controller]),
			command.NewController, helper.AsController(helper.Identity[*command.Controller]),
			github.NewController, helper.AsController(helper.Identity[*github.Controller]),
			basic.NewController, helper.AsController(helper.Identity[*basic.Controller]),
		),
		// opamp specific spec for connection context
		fx.Provide(
			func(opampController *opamp.Controller) func(context.Context, net.Conn) context.Context {
				return opampController.ConnContext
			},
		),
	)
}
