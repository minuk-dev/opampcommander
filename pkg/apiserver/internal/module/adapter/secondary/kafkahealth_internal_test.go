package secondary

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

func TestKafkaHealthIndicators_Registration(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		protocol config.EventProtocolType
		count    int
	}{
		{name: "kafka", protocol: config.EventProtocolTypeKafka, count: 1},
		{name: "inmemory", protocol: config.EventProtocolTypeInMemory},
		{name: "direct", protocol: config.EventProtocolTypeDirect},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			settings := &config.EventSettings{ProtocolType: test.protocol}
			indicators := newKafkaHealthIndicators(settings).Indicators
			require.Len(t, indicators, test.count)

			if test.count > 0 {
				require.Equal(t, "Kafka", indicators[0].Name())
			}
		})
	}
}
