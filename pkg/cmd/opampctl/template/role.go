package template

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed examples/role/viewer.yaml
var roleViewer string

//go:embed examples/role/agent-operator.yaml
var roleAgentOperator string

//go:embed examples/role/admin.yaml
var roleAdmin string

//nolint:gochecknoglobals // example registry for the role template command
var roleExamples = map[string]string{
	"viewer":         roleViewer,
	"agent-operator": roleAgentOperator,
	"admin":          roleAdmin,
}

func newRoleCommand() *cobra.Command {
	return newKindCommand("role", "Print a Role template", roleExamples)
}
