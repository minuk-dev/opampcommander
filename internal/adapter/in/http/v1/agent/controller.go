// Package agent provides domain models for the agent
package agent

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/api"
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
// @Failure  400 {object} ErrorModel
// @Failure  404 {object} ErrorModel
// @Failure  500 {object} ErrorModel
// @Router  /api/v1/agents/{id} [get].
//
//nolint:funlen // Get method is long due to detailed error handling and response construction.
func (c *Controller) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	baseURL := ctx.Request.URL.Scheme + "://" + ctx.Request.Host + ctx.Request.URL.Path

	instanceUID, err := uuid.Parse(id)
	if err != nil {
		c.logger.Error("failed to parse id", "error", err.Error())
		ctx.JSON(http.StatusBadRequest, api.ErrorModel{
			Type:     baseURL,
			Title:    "Invalid Instance UID",
			Status:   http.StatusBadRequest,
			Detail:   "The provided instance UID is not a valid UUID.",
			Instance: ctx.Request.URL.String(),
			Errors: []*api.ErrorDetail{
				{
					Message:  "invalid UUID format",
					Location: "path.id",
					Value:    id,
				},
			},
		})

		return
	}

	agent, err := c.agentUsecase.GetAgent(ctx.Request.Context(), instanceUID)
	if err != nil {
		if errors.Is(err, domainport.ErrResourceNotExist) {
			c.logger.Error("agent not found", "instanceUID", instanceUID.String())
			ctx.JSON(http.StatusNotFound, api.ErrorModel{
				Type:     baseURL,
				Title:    "Agent Not Found",
				Status:   http.StatusNotFound,
				Detail:   "No agent found with the provided instance UID.",
				Instance: ctx.Request.URL.String(),
				Errors: []*api.ErrorDetail{
					{
						Message:  "agent does not exist",
						Location: "path.id",
						Value:    id,
					},
				},
			})

			return
		}

		c.logger.Error("failed to get agent", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, api.ErrorModel{
			Type:     baseURL,
			Title:    "Internal Server Error",
			Status:   http.StatusInternalServerError,
			Detail:   "An error occurred while retrieving the agent.",
			Instance: ctx.Request.URL.String(),
			Errors: []*api.ErrorDetail{
				{
					Message:  err.Error(),
					Location: "server",
					Value:    nil,
				},
			},
		})

		return
	}

	ctx.JSON(http.StatusOK, agent)
}
