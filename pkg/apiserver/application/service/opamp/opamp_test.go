package opamp_test

import (
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/scheduler"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/opamp"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
)

var _ usecase.OpAMPUsecase = (*opamp.Service)(nil)
var _ scheduler.Scheduler = (*opamp.Service)(nil)
