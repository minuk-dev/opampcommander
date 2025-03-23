package agent

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/samber/lo"

	agentv1 "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

type Controller struct {
	logger *slog.Logger

	// usecases
	agentUsecase Usecase
}

type Usecase interface {
	port.GetAgentUsecase
	port.ListAgentUsecase
}

func NewController(usecase Usecase) *Controller {
	controller := &Controller{
		logger: slog.Default(),

		agentUsecase: usecase,
	}

	return controller
}

func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/v1/agents",
			Handler:     "http.v1.agent.List",
			HandlerFunc: c.List,
		},
		{
			Method:      "GET",
			Path:        "/v1/agents/:id",
			Handler:     "http.v1.agent.Get",
			HandlerFunc: c.Get,
		},
	}
}

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
		c.logger.Error("failed to get agent", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	ctx.JSON(http.StatusOK, agent)
}
