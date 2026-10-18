package template

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed examples/remoteconfigschema/basic.yaml
var remoteConfigSchemaBasic string

func newRemoteConfigSchemaCommand() *cobra.Command {
	return newKindCommand("remoteconfigschema", "Print a RemoteConfigSchema template",
		map[string]string{"basic": remoteConfigSchemaBasic})
}
