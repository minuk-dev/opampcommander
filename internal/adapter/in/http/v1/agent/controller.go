// Package agent provides domain models for the agent
package agent

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that implements the agent controller.
type Controller struct {
	logger *slog.Logger

	// usecases
	agentUsecase ManageUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase ManageUsecase,
	logger *slog.Logger,
) *Controller {
	controller := &Controller{
		logger:       logger,
		agentUsecase: usecase,
	}

	return controller
}

// RoutesInfo returns the routes information for the agent controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/agents",
			Handler:     "http.v1.agent.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/agents/search",
			Handler:     "http.v1.agent.Search",
			HandlerFunc: c.Search,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/agents/:id",
			Handler:     "http.v1.agent.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodPut,
			Path:        "/api/v1/agents/:id/new-instance-uid",
			Handler:     "http.v1.agent.SetNewInstanceUID",
			HandlerFunc: c.SetNewInstanceUID,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/agents/:id/restart",
			Handler:     "http.v1.agent.Restart",
			HandlerFunc: c.Restart,
		},
	}
}

// List retrieves a list of agents.
//
// @Summary  List Agents
// @Tags agent
// @Description Retrieve a list of agents.
// @Accept json
// @Produce json
// @Success 200 {array} Agent
// @Param limit query int false "Maximum number of agents to return"
// @Param continue query string false "Token to continue listing agents"
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/agents [get].
func (c *Controller) List(ctx *gin.Context) {
	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.agentUsecase.ListAgents(ctx.Request.Context(), &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		c.logger.Error("failed to list agents", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the list of agents.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Search searches agents by query string.
//
// @Summary  Search Agents
// @Tags agent
// @Description Search agents by instance UID query.
// @Accept json
// @Produce json
// @Success 200 {array} Agent
// @Param q query string true "Search query for instance UID"
// @Param limit query int false "Maximum number of agents to return"
// @Param continue query string false "Token to continue listing agents"
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/agents/search [get].
func (c *Controller) Search(ctx *gin.Context) {
	query := ctx.Query("q")
	if query == "" {
		ginutil.HandleValidationError(ctx, "q", "", ginutil.ErrRequiredParam, false)

		return
	}

	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.agentUsecase.SearchAgents(ctx.Request.Context(), query, &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		c.logger.Error("failed to search agents", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while searching agents.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves an agent by its instance UID.
//
// @Summary  Get Agent
// @Tags agent
// @Description Retrieve an agent by its instance UID.
// @Accept  json
// @Produce  json
// @Param  id path string true "Instance UID of the agent"
// @Success  200 {object} Agent
// @Failure  400 {object} ErrorModel
// @Failure  404 {object} ErrorModel
// @Failure  500 {object} ErrorModel
// @Router  /api/v1/agents/{id} [get].
func (c *Controller) Get(ctx *gin.Context) {
	instanceUID, err := ginutil.ParseUUID(ctx, "id")
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	agent, err := c.agentUsecase.GetAgent(ctx.Request.Context(), instanceUID)
	if err != nil {
		c.logger.Error("failed to get agent", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the agent.")

		return
	}

	ctx.JSON(http.StatusOK, agent)
}

// SetNewInstanceUID sets a new instance UID for an agent.
//
// @Summary  Set New Instance UID
// @Tags agent
// @Description Set a new instance UID for an agent.
// @Accept json
// @Produce json
// @Param id path string true "Instance UID of the agent"
// @Param request body SetNewInstanceUIDRequest true "New instance UID request"
// @Success 200 {object} Agent
// @Failure 400 {object} ErrorModel
// @Failure 404 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/agents/{id}/new-instance-uid [put].
func (c *Controller) SetNewInstanceUID(ctx *gin.Context) {
	instanceUID, err := ginutil.ParseUUID(ctx, "id")
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	var req v1agent.SetNewInstanceUIDRequest

	err = ctx.ShouldBindJSON(&req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, true)

		return
	}

	agent, err := c.agentUsecase.SetNewInstanceUID(ctx.Request.Context(), instanceUID, req.NewInstanceUID)
	if err != nil {
		c.logger.Error("failed to set new instance UID", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while setting the new instance UID.")

		return
	}

	ctx.JSON(http.StatusOK, agent)
}

// Restart restarts an agent by its instance UID.
//
// @Summary  Restart Agent
// @Tags agent
// @Description Restart an agent by its instance UID.
// @Accept  json
// @Produce  json
// @Param  id path string true "Instance UID of the agent"
// @Success  200 "Agent restarted successfully"
// @Failure  400 {object} ErrorModel
// @Failure  404 {object} ErrorModel
// @Failure  500 {object} ErrorModel
// @Router  /api/v1/agents/{id}/restart [post].
func (c *Controller) Restart(ctx *gin.Context) {
	instanceUID, err := ginutil.ParseUUID(ctx, "id")
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	err = c.agentUsecase.RestartAgent(ctx.Request.Context(), instanceUID)
	if err != nil {
		c.logger.Error("failed to restart agent", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while restarting the agent.")

		return
	}

	ctx.Status(http.StatusOK)
}
