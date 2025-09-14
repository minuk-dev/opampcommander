// Package agentgroup provides HTTP handlers for managing agent groups.
package agentgroup

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	agentgroupv1 "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
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
			Path:        "/api/v1/agentgroups/:uid",
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
			Path:        "/api/v1/agentgroups/:uid",
			Handler:     "http.v1.agentgroup.Update",
			HandlerFunc: c.Update,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/api/v1/agentgroups/:uid",
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
// @Success 200 {array} AgentGroup
// @Param limit query int false "Maximum number of agent groups to return"
// @Param continue query string false "Token to continue listing agent groups"
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentgroups [get].
func (c *Controller) List(ctx *gin.Context) {
	limit, err := ginutil.GetQueryInt64(ctx, "limit", 0)
	if err != nil {
		c.logger.Error("failed to get limit from query", slog.String("error", err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})

		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.agentGroupUsecase.ListAgentGroups(ctx, &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		c.logger.Error("failed to list agent groups", slog.String("error", err.Error()))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

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
// @Param uid path string true "Agent Group ID"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentgroups/{uid} [get].
func (c *Controller) Get(ctx *gin.Context) {
	id := ctx.Param("uid")

	uuid, err := uuid.Parse(id)
	if err != nil {
		c.logger.Error("failed to parse agent group ID", slog.String("error", err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent group ID"})

		return
	}

	agentGroup, err := c.agentGroupUsecase.GetAgentGroup(ctx, uuid)
	if err != nil {
		if errors.Is(err, domainport.ErrResourceNotExist) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})

			return
		}

		c.logger.Error("failed to get agent group", slog.String("error", err.Error()))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	ctx.JSON(http.StatusOK, agentGroup)
}

// Create creates a new agent group.
//
// @Summary Create Agent Group
// @Tags agentgroup
// @Description Create a new agent group.
// @Accept json
// @Produce json
// @Param agentGroup body AgentGroupCreateRequest true "Agent Group to create"
// @Success 201 {object} AgentGroup
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentgroups [post].
func (c *Controller) Create(ctx *gin.Context) {
	var req agentgroupv1.CreateRequest

	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		c.logger.Error("failed to bind request", slog.String("error", err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})

		return
	}

	created, err := c.agentGroupUsecase.CreateAgentGroup(ctx, &CreateAgentGroupCommand{
		Name:       req.Name,
		Attributes: req.Attributes,
		Selector:   req.Selector,
	})
	if err != nil {
		c.logger.Error("failed to create agent group", slog.String("error", err.Error()))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	ctx.Header("Location", "/api/v1/agentgroups/"+created.UID.String())
	ctx.JSON(http.StatusCreated, created)
}

// Update updates an existing agent group.
//
// @Summary Update Agent Group
// @Tags agentgroup
// @Description Update an existing agent group.
// @Accept json
// @Produce json
// @Param uid path string true "Agent Group ID"
// @Param agentGroup body AgentGroup true "Updated Agent Group"
// @Success 200 {object} AgentGroup
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentgroups/{uid} [put].
func (c *Controller) Update(ctx *gin.Context) {
	id := ctx.Param("uid")

	uid, err := uuid.Parse(id)
	if err != nil {
		c.logger.Error("failed to parse agent group ID", slog.String("error", err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent group ID"})

		return
	}

	var req agentgroupv1.AgentGroup

	err = ctx.ShouldBindJSON(&req)
	if err != nil {
		c.logger.Error("failed to bind request", slog.String("error", err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})

		return
	}

	updated, err := c.agentGroupUsecase.UpdateAgentGroup(ctx, uid, &req)
	if err != nil {
		if errors.Is(err, domainport.ErrResourceNotExist) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})

			return
		}

		c.logger.Error("failed to update agent group", slog.String("error", err.Error()))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// Delete marks an agent group as deleted.
//
// @Summary Delete Agent Group
// @Tags agentgroup
// @Description Mark an agent group as deleted.
// @Param uid path string true "Agent Group ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentgroups/{uid} [delete].
func (c *Controller) Delete(ctx *gin.Context) {
	id := ctx.Param("uid")

	uid, err := uuid.Parse(id)
	if err != nil {
		c.logger.Error("failed to parse agent group ID", slog.String("error", err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent group ID"})

		return
	}

	deletedBy := ctx.Query("deletedBy")

	err = c.agentGroupUsecase.DeleteAgentGroup(ctx, uid, deletedBy)
	if err != nil {
		if errors.Is(err, domainport.ErrResourceNotExist) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "agent group not found"})

			return
		}

		c.logger.Error("failed to delete agent group", slog.String("error", err.Error()))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	ctx.Status(http.StatusNoContent)
}
