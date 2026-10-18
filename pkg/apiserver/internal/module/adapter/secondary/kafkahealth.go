package secondary

import (
	"go.uber.org/fx"

	outkafka "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/messaging/kafka"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

type kafkaHealthIndicators struct {
	fx.Out

	Indicators []healthcheck.HealthIndicator `group:"health_indicators,flatten"`
}

func newKafkaHealthIndicators(settings *config.EventSettings) kafkaHealthIndicators {
	result := kafkaHealthIndicators{}
	if settings.ProtocolType == config.EventProtocolTypeKafka {
		result.Indicators = []healthcheck.HealthIndicator{
			outkafka.NewHealthIndicator(settings.KafkaSettings.Brokers),
		}
	}

	return result
}
