package template

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed examples/certificate/basic.yaml
var certificateBasic string

//nolint:gochecknoglobals // example registry for the certificate template command
var certificateExamples = map[string]string{
	"basic": certificateBasic,
}

func newCertificateCommand() *cobra.Command {
	return newKindCommand("certificate", "Print a Certificate template", certificateExamples)
}
