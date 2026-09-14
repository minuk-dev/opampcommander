package template

import (
	_ "embed"

	"github.com/spf13/cobra"
)

// basicExample is the name every resource's minimal template is registered
// under, so `template examples <resource> basic` means the same thing for all
// of them.
const basicExample = "basic"

//go:embed examples/agentgroup/basic.yaml
var agentGroupBasic string

//go:embed examples/agentgroup/with-remote-config.yaml
var agentGroupWithRemoteConfig string

//go:embed examples/agentgroup/with-config-ref.yaml
var agentGroupWithConfigRef string

//nolint:gochecknoglobals // example registry for the agentgroup template command
var agentGroupExamples = map[string]string{
	basicExample:         agentGroupBasic,
	"with-remote-config": agentGroupWithRemoteConfig,
	"with-config-ref":    agentGroupWithConfigRef,
}

func newAgentGroupCommand() *cobra.Command {
	return newKindCommand("agentgroup", "Print an AgentGroup template", agentGroupExamples)
}
