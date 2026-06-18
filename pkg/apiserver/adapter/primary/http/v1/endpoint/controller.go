// Package endpoint contains controller for endpoint endpoints.
package endpoint

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

// Controller is a struct that implements the endpoint controller.
type Controller struct {
	logger *slog.Logger

	endpointUsecase port.EndpointManageUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase port.EndpointManageUsecase,
	logger *slog.Logger,
) *Controller {
	controller := &Controller{
		logger:          logger,
		endpointUsecase: usecase,
	}

	return controller
}

// RoutesInfo returns the routes information for the endpoint controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace/endpoints",
			Handler:     "http.v1.endpoint.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace/endpoints/:name",
			Handler:     "http.v1.endpoint.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/namespaces/:namespace/endpoints",
			Handler:     "http.v1.endpoint.Create",
			HandlerFunc: c.Create,
		},
		{
			Method:      http.MethodPut,
			Path:        "/api/v1/namespaces/:namespace/endpoints/:name",
			Handler:     "http.v1.endpoint.Update",
			HandlerFunc: c.Update,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/api/v1/namespaces/:namespace/endpoints/:name",
			Handler:     "http.v1.endpoint.Delete",
			HandlerFunc: c.Delete,
		},
	}
}

// List retrieves a list of endpoints.
//
// @Summary  List Endpoints
// @Tags endpoint
// @Description Retrieve a list of endpoints in a namespace.
// @Success 200 {object} v1.ListResponse[v1.Endpoint]
// @Param namespace path string true "Namespace"
// @Param limit query int false "Maximum number of endpoints to return"
// @Param continue query string false "Token to continue listing endpoints"
// @Param includeDeleted query bool false "Include soft-deleted endpoints"
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/endpoints [get].
func (c *Controller) List(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "namespace", ctx.Param("namespace"), err, true,
		)

		return
	}

	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "limit", ctx.Query("limit"), err, false,
		)

		return
	}

	continueToken := ctx.Query("continue")

	includeDeleted, err := ginutil.ParseBool(ctx, "includeDeleted", false)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "includeDeleted", ctx.Query("includeDeleted"), err, false,
		)

		return
	}

	response, err := c.endpointUsecase.ListEndpoints(
		ctx.Request.Context(), namespace, &model.ListOptions{
			Limit:          limit,
			Continue:       continueToken,
			IncludeDeleted: includeDeleted,
		},
	)
	if err != nil {
		c.logger.Error(
			"failed to list endpoints", "error", err.Error(),
		)
		ginutil.InternalServerError(
			ctx, err,
			"An error occurred while retrieving endpoints.",
		)

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves an endpoint by its name.
//
// @Summary  Get Endpoint
// @Tags endpoint
// @Description Retrieve an endpoint by its name.
// @Success 200 {object} v1.Endpoint
// @Param namespace path string true "Namespace"
// @Param name path string true "Name of the endpoint"
// @Param includeDeleted query bool false "Include soft-deleted endpoint"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/endpoints/{name} [get].
func (c *Controller) Get(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "namespace", ctx.Param("namespace"), err, true,
		)

		return
	}

	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "name", ctx.Param("name"), err, true,
		)

		return
	}

	includeDeleted, err := ginutil.ParseBool(ctx, "includeDeleted", false)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "includeDeleted", ctx.Query("includeDeleted"), err, false,
		)

		return
	}

	endpoint, err := c.endpointUsecase.GetEndpoint(
		ctx.Request.Context(), namespace, name, &model.GetOptions{
			IncludeDeleted: includeDeleted,
		},
	)
	if err != nil {
		c.logger.Error(
			"failed to get endpoint",
			"name", name, "error", err.Error(),
		)
		ginutil.HandleDomainError(
			ctx, err,
			"An error occurred while retrieving the endpoint.",
		)

		return
	}

	ctx.JSON(http.StatusOK, endpoint)
}

// Create creates a new endpoint.
//
// @Summary  Create Endpoint
// @Tags endpoint
// @Description Create a new endpoint.
// @Accept json
// @Produce json
// @Success 201 {object} v1.Endpoint
// @Param namespace path string true "Namespace"
// @Param endpoint body v1.Endpoint true "Endpoint to create"
// @Failure 400 {object} map[string]any
// @Failure 409 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/endpoints [post].
func (c *Controller) Create(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "namespace", ctx.Param("namespace"), err, true,
		)

		return
	}

	var req v1.Endpoint

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	req.Metadata.Namespace = namespace

	created, err := c.endpointUsecase.CreateEndpoint(
		ctx.Request.Context(), &req,
	)
	if err != nil {
		c.logger.Error(
			"failed to create endpoint", "error", err.Error(),
		)
		ginutil.HandleDomainError(
			ctx, err,
			"An error occurred while creating the endpoint.",
		)

		return
	}

	ctx.Header(
		"Location",
		"/api/v1/namespaces/"+namespace+
			"/endpoints/"+created.Metadata.Name,
	)
	ctx.JSON(http.StatusCreated, created)
}

// Update updates an existing endpoint.
//
// @Summary  Update Endpoint
// @Tags endpoint
// @Description Update an existing endpoint.
// @Accept json
// @Produce json
// @Success 200 {object} v1.Endpoint
// @Param namespace path string true "Namespace"
// @Param name path string true "Name of the endpoint"
// @Param endpoint body v1.Endpoint true "Updated Endpoint"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/endpoints/{name} [put].
func (c *Controller) Update(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "namespace", ctx.Param("namespace"), err, true,
		)

		return
	}

	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "name", ctx.Param("name"), err, true,
		)

		return
	}

	var req v1.Endpoint

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	updated, err := c.endpointUsecase.UpdateEndpoint(
		ctx.Request.Context(), namespace, name, &req,
	)
	if err != nil {
		c.logger.Error(
			"failed to update endpoint",
			"name", name, "error", err.Error(),
		)
		ginutil.HandleDomainError(
			ctx, err,
			"An error occurred while updating the endpoint.",
		)

		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// Delete deletes an endpoint by its name.
//
// @Summary  Delete Endpoint
// @Tags endpoint
// @Description Delete an endpoint by its name.
// @Param namespace path string true "Namespace"
// @Param name path string true "Name of the endpoint"
// @Success 204
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/endpoints/{name} [delete].
func (c *Controller) Delete(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "namespace", ctx.Param("namespace"), err, true,
		)

		return
	}

	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "name", ctx.Param("name"), err, true,
		)

		return
	}

	err = c.endpointUsecase.DeleteEndpoint(
		ctx.Request.Context(), namespace, name,
	)
	if err != nil {
		c.logger.Error(
			"failed to delete endpoint",
			"name", name, "error", err.Error(),
		)
		ginutil.HandleDomainError(
			ctx, err,
			"An error occurred while deleting the endpoint.",
		)

		return
	}

	ctx.Status(http.StatusNoContent)
}
