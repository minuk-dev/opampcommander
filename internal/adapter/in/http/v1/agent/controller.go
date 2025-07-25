// Package agent provides domain models for the agent
package agent

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	agentv1 "github.com/minuk-dev/opampcommander/api/v1/agent"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that implements the agent controller.
type Controller struct {
	logger *slog.Logger

	// usecases
	agentUsecase applicationport.AgentManageUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase applicationport.AgentManageUsecase,
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
		{
			Method:      "POST",
			Path:        "/api/v1/agents/:id/update-agent-config",
			Handler:     "http.v1.agent.UpdateAgentConfig",
			HandlerFunc: c.UpdateAgentConfig,
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

	response, err := c.agentUsecase.ListAgents(ctx, &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		c.logger.Error("failed to list agents", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	agentResponse := agentv1.NewListResponse(
		lo.Map(response.Items, func(agent *model.Agent, _ int) agentv1.Agent {
			return agentv1.Agent{
				InstanceUID: agent.InstanceUID,
				Raw:         agent,
			}
		}),
		v1.ListMeta{
			RemainingItemCount: response.RemainingItemCount,
			Continue:           response.Continue,
		},
	)

	ctx.JSON(http.StatusOK, agentResponse)
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

	agent, err := c.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		if errors.Is(err, domainport.ErrAgentNotExist) {
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

// UpdateAgentConfig creates a new command to update the agent configuration.
//
// @Summary  Update Agent Configuration
// @Tags agent
// @Description Create a new command to update the agent configuration.
// @Accept  json
// @Produce  json
// @Param  id path string true "Instance UID of the agent"
// @Param  request body UpdateAgentConfigRequest true "Request body containing the remote configuration"
// @Success  201 {object} AgentCommand
// @Failure  400 {object} map[string]any
// @Failure  500 {object} map[string]any
// @Router  /api/v1/agents/{id}/update-agent-config [post].
func (c *Controller) UpdateAgentConfig(ctx *gin.Context) {
	id := ctx.Param("id")

	instanceUID, err := uuid.Parse(id)
	if err != nil {
		c.logger.Error("failed to parse id", "error", err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	var request agentv1.UpdateAgentConfigRequest

	err = ctx.ShouldBindJSON(&request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})

		return
	}

	command := model.NewUpdateAgentConfigCommand(instanceUID, request.RemoteConfig)

	err = c.agentUsecase.SendCommand(ctx, instanceUID, command)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save command"})

		return
	}

	ctx.JSON(http.StatusCreated, convertToAPIModel(command))
}

func convertToAPIModel(command *model.Command) *agentv1.Command {
	return &agentv1.Command{
		Kind:              string(command.Kind),
		ID:                command.ID.String(),
		TargetInstanceUID: command.TargetInstanceUID.String(),
		Data:              command.Data,
	}
}
