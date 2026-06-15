// Package host contains controller for host related endpoints.
package host

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

// Controller is a struct that implements the host controller.
type Controller struct {
	logger      *slog.Logger
	hostUsecase ManageUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase ManageUsecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		logger:      logger,
		hostUsecase: usecase,
	}
}

// RoutesInfo returns the routes information for the host controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/hosts",
			Handler:     "http.v1.host.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/hosts/:id",
			Handler:     "http.v1.host.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/hosts/:id/agents",
			Handler:     "http.v1.host.ListAgents",
			HandlerFunc: c.ListAgents,
		},
	}
}

// List retrieves a list of hosts.
//
// @Summary  List Hosts
// @Tags host
// @Description Retrieve a list of discovered hosts.
// @Accept json
// @Produce json
// @Success 200 {object} v1.ListResponse[v1.Host]
// @Param limit query int false "Maximum number of hosts to return"
// @Param continue query string false "Token to continue listing hosts"
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/hosts [get].
func (c *Controller) List(ctx *gin.Context) {
	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	var response *v1.ListResponse[v1.Host]

	response, err = c.hostUsecase.ListHosts(
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
		c.logger.Error("failed to list hosts", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while retrieving hosts.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves a host by ID.
//
// @Summary  Get Host
// @Tags host
// @Description Retrieve a discovered host by its ID.
// @Accept json
// @Produce json
// @Success 200 {object} v1.Host
// @Param id path string true "Host ID"
// @Failure 400 {object} ErrorModel
// @Failure 404 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/hosts/{id} [get].
func (c *Controller) Get(ctx *gin.Context) {
	id, err := ginutil.ParseString(ctx, "id", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	var host *v1.Host

	host, err = c.hostUsecase.GetHost(ctx.Request.Context(), id)
	if err != nil {
		c.logger.Error("failed to get host", "id", id, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the host.")

		return
	}

	ctx.JSON(http.StatusOK, host)
}

// ListAgents retrieves the agents associated with a host.
//
// @Summary  List Host Agents
// @Tags host
// @Description Retrieve the agents running on a discovered host.
// @Accept json
// @Produce json
// @Success 200 {object} v1.ListResponse[v1.Agent]
// @Param id path string true "Host ID"
// @Param limit query int false "Maximum number of agents to return"
// @Param continue query string false "Token to continue listing agents"
// @Failure 400 {object} ErrorModel
// @Failure 404 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/hosts/{id}/agents [get].
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

	response, err = c.hostUsecase.ListAgentsByHost(
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
		c.logger.Error("failed to list host agents", "id", id, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the host agents.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}
