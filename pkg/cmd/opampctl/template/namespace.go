package template

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed examples/namespace/basic.yaml
var namespaceBasic string

//go:embed examples/namespace/with-labels.yaml
var namespaceWithLabels string

//nolint:gochecknoglobals // example registry for the namespace template command
var namespaceExamples = map[string]string{
	"basic":       namespaceBasic,
	"with-labels": namespaceWithLabels,
}

func newNamespaceCommand() *cobra.Command {
	return newKindCommand("namespace", "Print a Namespace template", namespaceExamples)
}
