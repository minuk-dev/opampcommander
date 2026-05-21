package template

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed examples/agentgroup/basic.yaml
var agentGroupBasic string

//go:embed examples/agentgroup/with-remote-config.yaml
var agentGroupWithRemoteConfig string

//go:embed examples/agentgroup/with-config-ref.yaml
var agentGroupWithConfigRef string

//nolint:gochecknoglobals // example registry for the agentgroup template command
var agentGroupExamples = map[string]string{
	"basic":              agentGroupBasic,
	"with-remote-config": agentGroupWithRemoteConfig,
	"with-config-ref":    agentGroupWithConfigRef,
}

func newAgentGroupCommand() *cobra.Command {
	return newKindCommand("agentgroup", "Print an AgentGroup template", agentGroupExamples)
}
