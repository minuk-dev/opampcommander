package opamp_test

import (
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/application/service/opamp"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
)

var _ applicationport.OpAMPUsecase = (*opamp.Service)(nil)
var _ helper.Runner = (*opamp.Service)(nil)
