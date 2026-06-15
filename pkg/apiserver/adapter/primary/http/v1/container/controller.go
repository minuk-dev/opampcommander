// Package container contains controller for container related endpoints.
package container

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

// Controller is a struct that implements the container controller.
type Controller struct {
	logger           *slog.Logger
	containerUsecase ManageUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase ManageUsecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		logger:           logger,
		containerUsecase: usecase,
	}
}

// RoutesInfo returns the routes information for the container controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/containers",
			Handler:     "http.v1.container.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/containers/:id",
			Handler:     "http.v1.container.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/containers/:id/agents",
			Handler:     "http.v1.container.ListAgents",
			HandlerFunc: c.ListAgents,
		},
	}
}

// List retrieves a list of containers.
//
// @Summary  List Containers
// @Tags container
// @Description Retrieve a list of discovered containers.
// @Accept json
// @Produce json
// @Success 200 {object} v1.ListResponse[v1.Container]
// @Param limit query int false "Maximum number of containers to return"
// @Param continue query string false "Token to continue listing containers"
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/containers [get].
func (c *Controller) List(ctx *gin.Context) {
	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	var response *v1.ListResponse[v1.Container]

	response, err = c.containerUsecase.ListContainers(
		ctx.Request.Context(),
		&model.ListOptions{
			Limit:                    limit,
			Continue:                 ctx.Query("continue"),
			IncludeDeleted:           false,
			ConnectedOnly:            false,
			IdentifyingAttributes:    nil,
			NonIdentifyingAttributes: nil,
		},
	)
	if err != nil {
		c.logger.Error("failed to list containers", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while retrieving containers.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves a container by ID.
//
// @Summary  Get Container
// @Tags container
// @Description Retrieve a discovered container by its ID.
// @Accept json
// @Produce json
// @Success 200 {object} v1.Container
// @Param id path string true "Container ID"
// @Failure 400 {object} ErrorModel
// @Failure 404 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/containers/{id} [get].
func (c *Controller) Get(ctx *gin.Context) {
	id, err := ginutil.ParseString(ctx, "id", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	var container *v1.Container

	container, err = c.containerUsecase.GetContainer(ctx.Request.Context(), id)
	if err != nil {
		c.logger.Error("failed to get container", "id", id, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the container.")

		return
	}

	ctx.JSON(http.StatusOK, container)
}

// ListAgents retrieves the agents associated with a container.
//
// @Summary  List Container Agents
// @Tags container
// @Description Retrieve the agents running in a discovered container.
// @Accept json
// @Produce json
// @Success 200 {object} v1.ListResponse[v1.Agent]
// @Param id path string true "Container ID"
// @Param limit query int false "Maximum number of agents to return"
// @Param continue query string false "Token to continue listing agents"
// @Failure 400 {object} ErrorModel
// @Failure 404 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/containers/{id}/agents [get].
func (c *Controller) ListAgents(ctx *gin.Context) {
	id, err := ginutil.ParseString(ctx, "id", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	var response *v1.ListResponse[v1.Agent]

	response, err = c.containerUsecase.ListAgentsByContainer(
		ctx.Request.Context(),
		id,
		&model.ListOptions{
			Limit:                    limit,
			Continue:                 ctx.Query("continue"),
			IncludeDeleted:           false,
			ConnectedOnly:            false,
			IdentifyingAttributes:    nil,
			NonIdentifyingAttributes: nil,
		},
	)
	if err != nil {
		c.logger.Error("failed to list container agents", "id", id, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the container agents.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}
