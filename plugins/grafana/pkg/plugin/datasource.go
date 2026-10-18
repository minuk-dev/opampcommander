package plugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
)

var (
	_ backend.QueryDataHandler   = (*Datasource)(nil)
	_ backend.CheckHealthHandler = (*Datasource)(nil)
)

const previewMessage = "Connections and fleet queries are not available in this preview."

type Datasource struct{}

func NewDatasource(context.Context, backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &Datasource{}, nil
}

// QueryData fails explicitly until authenticated read queries are implemented.
// An empty success or generated sample series would misrepresent fleet health.
func (d *Datasource) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()
	for _, query := range req.Queries {
		response.Responses[query.RefID] = backend.ErrDataResponse(backend.StatusNotImplemented, previewMessage)
	}
	return response, nil
}

func (d *Datasource) CheckHealth(context.Context, *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusError,
		Message: previewMessage,
	}, nil
}
