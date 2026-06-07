package opamp_test

import (
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/in/scheduler"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/opamp"
)

var _ applicationport.OpAMPUsecase = (*opamp.Service)(nil)
var _ scheduler.Scheduler = (*opamp.Service)(nil)
