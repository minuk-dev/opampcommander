// Package agentgroup provides HTTP handlers for managing agent groups.
package agentgroup

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that implements the agent group controller.
type Controller struct {
	logger *slog.Logger

	// usecases
	agentGroupUsecase Usecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase Usecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		logger:            logger,
		agentGroupUsecase: usecase,
	}
}

// RoutesInfo returns the routes information for the agent group controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/agentgroups",
			Handler:     "http.v1.agentgroup.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/agentgroups/:name/agents",
			Handler:     "http.v1.agentgroup.GetAgentByAgentGroup",
			HandlerFunc: c.ListAgentsByAgentGroup,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/agentgroups/:name",
			Handler:     "http.v1.agentgroup.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/agentgroups",
			Handler:     "http.v1.agentgroup.Create",
			HandlerFunc: c.Create,
		},
		{
			Method:      http.MethodPut,
			Path:        "/api/v1/agentgroups/:name",
			Handler:     "http.v1.agentgroup.Update",
			HandlerFunc: c.Update,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/api/v1/agentgroups/:name",
			Handler:     "http.v1.agentgroup.Delete",
			HandlerFunc: c.Delete,
		},
	}
}

// List retrieves a list of agent groups.
//
// @Summary List Agent Groups
// @Tags agentgroup
// @Description Retrieves a list of agent groups with pagination options.
// @Success 200 {object} v1.ListResponse[v1.AgentGroup]
// @Param limit query int false "Maximum number of agent groups to return"
// @Param continue query string false "Token to continue listing agent groups"
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentgroups [get].
func (c *Controller) List(ctx *gin.Context) {
	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.agentGroupUsecase.ListAgentGroups(ctx.Request.Context(), &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		c.logger.Error("failed to list agent groups", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while retrieving the list of agent groups.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves an agent group by its ID.
//
// @Summary Get Agent Group
// @Tags agentgroup
// @Description Retrieve an agent group by its ID.
// @Accept json
// @Produce json
// @Success 200 {object} AgentGroup
// @Param name path string true "Agent Group Name"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentgroups/{name} [get].
func (c *Controller) Get(ctx *gin.Context) {
	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "name", ctx.Param("name"), err, true)

		return
	}

	agentGroup, err := c.agentGroupUsecase.GetAgentGroup(ctx.Request.Context(), name)
	if err != nil {
		c.logger.Error("failed to get agent group", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the agent group.")

		return
	}

	ctx.JSON(http.StatusOK, agentGroup)
}

// ListAgentsByAgentGroup retrieves agents belonging to a specific agent group.
//
// @Summary List Agents by Agent Group
// @Tags agentgroup
// @Description Retrieve agents belonging to a specific agent group.
// @Accept json
// @Produce json
// @Success 200 {object} v1.ListResponse[v1.Agent]
// @Param name path string true "Agent Group Name".
func (c *Controller) ListAgentsByAgentGroup(ctx *gin.Context) {
	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	continueToken := ctx.Query("continue")

	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "name", ctx.Param("name"), err, true)

		return
	}

	agents, err := c.agentGroupUsecase.ListAgentsByAgentGroup(ctx.Request.Context(), name, &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		c.logger.Error("failed to get agents by agent group", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the agents for the agent group.")

		return
	}

	ctx.JSON(http.StatusOK, agents)
}

// Create creates a new agent group.
//
// @Summary Create Agent Group
// @Tags agentgroup
// @Description Create a new agent group.
// @Accept json
// @Produce json
// @Param agentGroup body AgentGroup true "Agent Group to create"
// @Success 201 {object} AgentGroup
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/agentgroups [post].
func (c *Controller) Create(ctx *gin.Context) {
	var req v1.AgentGroup

	err := ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	created, err := c.agentGroupUsecase.CreateAgentGroup(ctx.Request.Context(), &req)
	if err != nil {
		c.logger.Error("failed to create agent group", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while creating the agent group.")

		return
	}

	ctx.Header("Location", "/api/v1/agentgroups/"+created.Metadata.Name)
	ctx.JSON(http.StatusCreated, created)
}

// Update updates an existing agent group.
//
// @Summary Update Agent Group
// @Tags agentgroup
// @Description Update an existing agent group.
// @Accept json
// @Produce json
// @Param name path string true "Agent Group Name"
// @Param agentGroup body AgentGroup true "Updated Agent Group"
// @Success 200 {object} AgentGroup
// @Failure 400 {object} ErrorModel
// @Failure 404 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/agentgroups/{name} [put].
func (c *Controller) Update(ctx *gin.Context) {
	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "name", ctx.Param("name"), err, true)

		return
	}

	var req v1.AgentGroup

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	updated, err := c.agentGroupUsecase.UpdateAgentGroup(ctx.Request.Context(), name, &req)
	if err != nil {
		c.logger.Error("failed to update agent group", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while updating the agent group.")

		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// Delete marks an agent group as deleted.
//
// @Summary Delete Agent Group
// @Tags agentgroup
// @Description Mark an agent group as deleted.
// @Param name path string true "Agent Group ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorModel
// @Failure 404 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/agentgroups/{name} [delete].
func (c *Controller) Delete(ctx *gin.Context) {
	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "name", ctx.Param("name"), err, true)

		return
	}

	err = c.agentGroupUsecase.DeleteAgentGroup(ctx.Request.Context(), name)
	if err != nil {
		c.logger.Error("failed to delete agent group", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while deleting the agent group.")

		return
	}

	ctx.Status(http.StatusNoContent)
}
