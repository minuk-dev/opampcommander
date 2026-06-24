// Package endpointmetrics contains the controller for endpoint throughput.
package endpointmetrics

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

// Controller exposes endpoint-throughput queries over HTTP.
type Controller struct {
	logger *slog.Logger

	usecase port.EndpointMetricsUsecase
}

// NewController creates a new endpoint-throughput Controller.
func NewController(
	usecase port.EndpointMetricsUsecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		logger:  logger,
		usecase: usecase,
	}
}

// RoutesInfo returns the routes for the endpoint-throughput controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace/endpoint-throughputs",
			Handler:     "http.v1.endpointmetrics.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace/endpoints/:name/throughput",
			Handler:     "http.v1.endpointmetrics.Get",
			HandlerFunc: c.Get,
		},
	}
}

// Get reports the throughput of a single endpoint.
//
// It is declared before List so the non-generic v1.EndpointThroughput response
// type is registered with swag before List references it through the generic
// v1.ListResponse[v1.EndpointThroughput] (swag resolves generic instantiations
// in source order within a file).
//
// @Summary  Get Endpoint Throughput
// @Tags endpoint
// @Description Report how much telemetry collectors are sending to a single endpoint.
// @Success 200 {object} v1.EndpointThroughput
// @Param namespace path string true "Namespace"
// @Param name path string true "Name of the endpoint"
// @Param window query string false "Rate window as a Go duration (e.g. 5m); defaults to the server's configured window"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/endpoints/{name}/throughput [get].
func (c *Controller) Get(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "namespace", ctx.Param("namespace"), err, true)

		return
	}

	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "name", ctx.Param("name"), err, true)

		return
	}

	window, ok := parseWindow(ctx)
	if !ok {
		return
	}

	var response *v1.EndpointThroughput

	response, err = c.usecase.GetEndpointThroughput(ctx.Request.Context(), namespace, name, window)
	if err != nil {
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving endpoint throughput.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// List reports the throughput of every endpoint in a namespace.
//
// @Summary  List Endpoint Throughput
// @Tags endpoint
// @Description Report how much telemetry collectors are sending to each endpoint in a namespace.
// @Success 200 {object} v1.ListResponse[v1.EndpointThroughput]
// @Param namespace path string true "Namespace"
// @Param window query string false "Rate window as a Go duration (e.g. 5m); defaults to the server's configured window"
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/endpoint-throughputs [get].
func (c *Controller) List(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "namespace", ctx.Param("namespace"), err, true)

		return
	}

	window, ok := parseWindow(ctx)
	if !ok {
		return
	}

	var response *v1.ListResponse[v1.EndpointThroughput]

	response, err = c.usecase.ListEndpointThroughput(ctx.Request.Context(), namespace, window)
	if err != nil {
		c.logger.Error("failed to list endpoint throughput", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while retrieving endpoint throughput.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// parseWindow reads the optional "window" query parameter as a Go duration. An
// absent parameter yields 0 (the service applies its default); an invalid one
// writes a 400 and returns ok=false.
func parseWindow(ctx *gin.Context) (time.Duration, bool) {
	raw := ctx.Query("window")
	if raw == "" {
		return 0, true
	}

	window, err := time.ParseDuration(raw)
	if err != nil {
		ginutil.InvalidQueryParamError(ctx, "window", raw, "must be a valid duration (e.g. 5m)")

		return 0, false
	}

	return window, true
}
