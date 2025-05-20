// Package agent provides domain models for the agent
package agent

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/samber/lo"

	agentv1 "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

// Controller is a struct that implements the agent controller.
type Controller struct {
	logger *slog.Logger

	// usecases
	agentUsecase port.AgentUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase port.AgentUsecase,
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
			Method:      "GET",
			Path:        "/api/v1/agents",
			Handler:     "http.v1.agent.List",
			HandlerFunc: c.List,
		},
		{
			Method:      "GET",
			Path:        "/api/v1/agents/:id",
			Handler:     "http.v1.agent.Get",
			HandlerFunc: c.Get,
		},
	}
}

// List retrieves a list of agents.
func (c *Controller) List(ctx *gin.Context) {
	agents, err := c.agentUsecase.ListAgents(ctx)
	if err != nil {
		c.logger.Error("failed to list agents", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	agentResponse := lo.Map(agents, func(agent *model.Agent, _ int) *agentv1.Agent {
		return &agentv1.Agent{
			InstanceUID: agent.InstanceUID,
			Raw:         agent,
		}
	})

	ctx.JSON(http.StatusOK, agentResponse)
}

// Get retrieves an agent by its instance UID.
func (c *Controller) Get(ctx *gin.Context) {
	id := ctx.Param("id")

	instanceUID, err := uuid.Parse(id)
	if err != nil {
		c.logger.Error("failed to parse id", "error", err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	agent, err := c.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		if errors.Is(err, port.ErrAgentNotExist) {
			c.logger.Error("agent not found", "instanceUID", instanceUID.String())
			ctx.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})

			return
		}

		c.logger.Error("failed to get agent", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	ctx.JSON(http.StatusOK, &agentv1.Agent{
		InstanceUID: agent.InstanceUID,
		Raw:         agent,
	})
}
