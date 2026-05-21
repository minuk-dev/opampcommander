package template

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed examples/agentremoteconfig/otlp-debug.yaml
var agentRemoteConfigOTLPDebug string

//go:embed examples/agentremoteconfig/otlp-forward.yaml
var agentRemoteConfigOTLPForward string

//go:embed examples/agentremoteconfig/hostmetrics.yaml
var agentRemoteConfigHostmetrics string

//go:embed examples/agentremoteconfig/prometheus-scrape.yaml
var agentRemoteConfigPrometheusScrape string

//go:embed examples/agentremoteconfig/filelog.yaml
var agentRemoteConfigFilelog string

//go:embed examples/agentremoteconfig/kubernetes-attributes.yaml
var agentRemoteConfigK8sAttributes string

//go:embed examples/agentremoteconfig/opamp-extension.yaml
var agentRemoteConfigOpAMPExtension string

//nolint:gochecknoglobals // example registry for the agentremoteconfig template command
var agentRemoteConfigExamples = map[string]string{
	"otlp-debug":            agentRemoteConfigOTLPDebug,
	"otlp-forward":          agentRemoteConfigOTLPForward,
	"hostmetrics":           agentRemoteConfigHostmetrics,
	"prometheus-scrape":     agentRemoteConfigPrometheusScrape,
	"filelog":               agentRemoteConfigFilelog,
	"kubernetes-attributes": agentRemoteConfigK8sAttributes,
	"opamp-extension":       agentRemoteConfigOpAMPExtension,
}

func newAgentRemoteConfigCommand() *cobra.Command {
	return newKindCommand("agentremoteconfig", "Print an AgentRemoteConfig template", agentRemoteConfigExamples)
}
