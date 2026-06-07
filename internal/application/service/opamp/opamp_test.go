package opamp_test

import (
	"github.com/minuk-dev/opampcommander/internal/adapter/in/scheduler"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/application/service/opamp"
)

var _ applicationport.OpAMPUsecase = (*opamp.Service)(nil)
var _ scheduler.Scheduler = (*opamp.Service)(nil)
