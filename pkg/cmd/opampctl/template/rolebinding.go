package template

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed examples/rolebinding/user.yaml
var roleBindingUser string

//nolint:gochecknoglobals // example registry for the rolebinding template command
var roleBindingExamples = map[string]string{
	"user": roleBindingUser,
}

func newRoleBindingCommand() *cobra.Command {
	return newKindCommand("rolebinding", "Print a RoleBinding template", roleBindingExamples)
}
