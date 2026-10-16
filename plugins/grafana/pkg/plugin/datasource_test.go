package plugin_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"

	"github.com/minuk-dev/opampcommander/plugins/grafana/pkg/plugin"
)

func TestQueryDataDoesNotInventFleetData(t *testing.T) {
	t.Parallel()

	instance, err := plugin.NewDatasource(t.Context(), backend.DataSourceInstanceSettings{})
	if err != nil {
		t.Fatal(err)
	}
	response, err := instance.(backend.QueryDataHandler).QueryData(t.Context(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{{RefID: "A"}, {RefID: "B"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Responses) != 2 {
		t.Fatalf("got %d responses, want one error per query", len(response.Responses))
	}
	for _, refID := range []string{"A", "B"} {
		result := response.Responses[refID]
		if result.Error == nil || result.Status != backend.StatusNotImplemented || len(result.Frames) != 0 {
			t.Errorf("query %s must return not implemented without frames: %+v", refID, result)
		}
	}
}

func TestCheckHealthDoesNotClaimConnectivity(t *testing.T) {
	t.Parallel()

	instance, err := plugin.NewDatasource(t.Context(), backend.DataSourceInstanceSettings{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := instance.(backend.CheckHealthHandler).CheckHealth(t.Context(), &backend.CheckHealthRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != backend.HealthStatusError || result.Message == "" {
		t.Fatalf("unimplemented connection must report an explanatory health error: %+v", result)
	}
}
