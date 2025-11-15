// Package agent provides domain models for the agent
package agent

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
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
		logger: logger,

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
			Path:        "/api/v1/agents/:id",
			Handler:     "http.v1.agent.Get",
			HandlerFunc: c.Get,
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
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agents [get].
func (c *Controller) List(ctx *gin.Context) {
	limit, err := ginutil.GetQueryInt64(ctx, "limit", 0)
	if err != nil {
		c.logger.Error("failed to parse limit", "error", err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})

		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.agentUsecase.ListAgents(ctx.Request.Context(), &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		c.logger.Error("failed to list agents", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

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
// @Failure  400 {object} map[string]any
// @Failure  404 {object} map[string]any
// @Failure  500 {object} map[string]any
// @Router  /api/v1/agents/{id} [get].
func (c *Controller) Get(ctx *gin.Context) {
	id := ctx.Param("id")

	instanceUID, err := uuid.Parse(id)
	if err != nil {
		c.logger.Error("failed to parse id", "error", err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	agent, err := c.agentUsecase.GetAgent(ctx.Request.Context(), instanceUID)
	if err != nil {
		if errors.Is(err, domainport.ErrResourceNotExist) {
			c.logger.Error("agent not found", "instanceUID", instanceUID.String())
			ctx.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})

			return
		}

		c.logger.Error("failed to get agent", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	ctx.JSON(http.StatusOK, agent)
}
