package service_test

import (
	"github.com/minuk-dev/opampcommander/internal/domain/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
)

var _ helper.Runner = (*service.AgentGroupService)(nil)
