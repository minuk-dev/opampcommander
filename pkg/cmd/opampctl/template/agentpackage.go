package template

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed examples/agentpackage/top-level.yaml
var agentPackageTopLevel string

//go:embed examples/agentpackage/addon.yaml
var agentPackageAddon string

//nolint:gochecknoglobals // example registry for the agentpackage template command
var agentPackageExamples = map[string]string{
	"top-level": agentPackageTopLevel,
	"addon":     agentPackageAddon,
}

func newAgentPackageCommand() *cobra.Command {
	return newKindCommand("agentpackage", "Print an AgentPackage template", agentPackageExamples)
}
