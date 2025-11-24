// Package agentgroup provides HTTP handlers for managing agent groups.
package agentgroup

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/minuk-dev/opampcommander/api"
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
// @Success 200 {array} AgentGroup
// @Param limit query int false "Maximum number of agent groups to return"
// @Param continue query string false "Token to continue listing agent groups"
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentgroups [get].
func (c *Controller) List(ctx *gin.Context) {
	baseURL := ginutil.GetErrorTypeURI(ctx)

	limit, err := ginutil.GetQueryInt64(ctx, "limit", 0)
	if err != nil {
		c.logger.Error("failed to get limit from query", slog.String("error", err.Error()))
		ctx.JSON(http.StatusBadRequest, &api.ErrorModel{
			Type:     baseURL,
			Title:    "Invalid Query Parameter",
			Status:   http.StatusBadRequest,
			Detail:   "The 'limit' query parameter must be a valid integer.",
			Instance: ctx.Request.URL.String(),
			Errors: []*api.ErrorDetail{
				{
					Message:  "must be a valid integer",
					Location: "query.limit",
					Value:    ctx.Query("limit"),
				},
			},
		})

		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.agentGroupUsecase.ListAgentGroups(ctx.Request.Context(), &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		c.logger.Error("failed to list agent groups", slog.String("error", err.Error()))
		ctx.JSON(http.StatusInternalServerError, api.ErrorModel{
			Type:     baseURL,
			Title:    "Internal Server Error",
			Status:   http.StatusInternalServerError,
			Detail:   "An error occurred while retrieving the list of agent groups.",
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
	baseURL := ginutil.GetErrorTypeURI(ctx)
	name := ctx.Param("name")

	agentGroup, err := c.agentGroupUsecase.GetAgentGroup(ctx.Request.Context(), name)
	if err != nil {
		if errors.Is(err, domainport.ErrResourceNotExist) {
			ctx.JSON(http.StatusNotFound, &api.ErrorModel{
				Type:     baseURL,
				Title:    "Not Found",
				Status:   http.StatusNotFound,
				Detail:   "The requested agent group does not exist.",
				Instance: ctx.Request.URL.String(),
				Errors: []*api.ErrorDetail{
					{
						Message:  "agent group not found",
						Location: "path.name",
						Value:    name,
					},
				},
			})

			return
		}

		c.logger.Error("failed to get agent group", slog.String("error", err.Error()))
		ctx.JSON(http.StatusInternalServerError, &api.ErrorModel{
			Type:     baseURL,
			Title:    "Internal Server Error",
			Status:   http.StatusInternalServerError,
			Detail:   "An error occurred while retrieving the agent group.",
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

	ctx.JSON(http.StatusOK, agentGroup)
}

// ListAgentsByAgentGroup retrieves agents belonging to a specific agent group.
//
// @Summary List Agents by Agent Group
// @Tags agentgroup
// @Description Retrieve agents belonging to a specific agent group.
// @Accept json
// @Produce json
// @Success 200 {array} Agent
// @Param name path string true "Agent Group Name".
func (c *Controller) ListAgentsByAgentGroup(gCtx *gin.Context) {
	baseURL := ginutil.GetErrorTypeURI(gCtx)

	limit, err := ginutil.GetQueryInt64(gCtx, "limit", 0)
	if err != nil {
		c.logger.Error("failed to get limit from query", slog.String("error", err.Error()))
		gCtx.JSON(http.StatusBadRequest, &api.ErrorModel{
			Type:     baseURL,
			Title:    "Invalid Query Parameter",
			Status:   http.StatusBadRequest,
			Detail:   "The 'limit' query parameter must be a valid integer.",
			Instance: gCtx.Request.URL.String(),
			Errors: []*api.ErrorDetail{
				{
					Message:  "must be a valid integer",
					Location: "query.limit",
					Value:    gCtx.Query("limit"),
				},
			},
		})

		return
	}

	continueToken := gCtx.Query("continue")
	name := gCtx.Param("name")

	agent, err := c.agentGroupUsecase.ListAgentsByAgentGroup(gCtx.Request.Context(), name, &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		if errors.Is(err, domainport.ErrResourceNotExist) {
			gCtx.JSON(http.StatusNotFound, &api.ErrorModel{
				Type:     baseURL,
				Title:    "Not Found",
				Status:   http.StatusNotFound,
				Detail:   "The requested agent group does not exist.",
				Instance: gCtx.Request.URL.String(),
				Errors: []*api.ErrorDetail{
					{
						Message:  "agent group not found",
						Location: "path.name",
						Value:    name,
					},
				},
			})

			return
		}

		c.logger.Error("failed to get agents by agent group", slog.String("error", err.Error()))
		gCtx.JSON(http.StatusInternalServerError, &api.ErrorModel{
			Type:     baseURL,
			Title:    "Internal Server Error",
			Status:   http.StatusInternalServerError,
			Detail:   "An error occurred while retrieving the agents for the agent group.",
			Instance: gCtx.Request.URL.String(),
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

	gCtx.JSON(http.StatusOK, agent)
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
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/agentgroups [post].
func (c *Controller) Create(ctx *gin.Context) {
	baseURL := ginutil.GetErrorTypeURI(ctx)

	var req agentgroupv1.CreateRequest

	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		c.logger.Error("failed to bind request", slog.String("error", err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})

		return
	}

	created, err := c.agentGroupUsecase.CreateAgentGroup(ctx.Request.Context(), &CreateAgentGroupCommand{
		Name:        req.Name,
		Priority:    req.Priority,
		Attributes:  req.Attributes,
		Selector:    req.Selector,
		AgentConfig: req.AgentConfig,
	})
	if err != nil {
		c.logger.Error("failed to create agent group", slog.String("error", err.Error()))
		ctx.JSON(http.StatusInternalServerError, &api.ErrorModel{
			Type:     baseURL,
			Title:    "Internal Server Error",
			Status:   http.StatusInternalServerError,
			Detail:   "An error occurred while creating the agent group.",
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

	ctx.Header("Location", "/api/v1/agentgroups/"+created.Name)
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
	baseURL := ginutil.GetErrorTypeURI(ctx)
	name := ctx.Param("name")

	var req agentgroupv1.AgentGroup

	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		c.logger.Error("failed to bind request", slog.String("error", err.Error()))
		ctx.JSON(http.StatusBadRequest, &api.ErrorModel{
			Type:     baseURL,
			Title:    "Invalid Request Body",
			Status:   http.StatusBadRequest,
			Detail:   "The request body is not valid JSON or does not conform to the expected schema.",
			Instance: ctx.Request.URL.String(),
			Errors: []*api.ErrorDetail{
				{
					Message:  err.Error(),
					Location: "body",
					Value:    nil,
				},
			},
		})

		return
	}

	updated, err := c.agentGroupUsecase.UpdateAgentGroup(ctx.Request.Context(), name, &req)
	if err != nil {
		if errors.Is(err, domainport.ErrResourceNotExist) {
			ctx.JSON(http.StatusNotFound, &api.ErrorModel{
				Type:     baseURL,
				Title:    "Not Found",
				Status:   http.StatusNotFound,
				Detail:   "The agent group to update does not exist.",
				Instance: ctx.Request.URL.String(),
				Errors: []*api.ErrorDetail{
					{
						Message:  "agent group not found",
						Location: "path.name",
						Value:    name,
					},
				},
			})

			return
		}

		c.logger.Error("failed to update agent group", slog.String("error", err.Error()))
		ctx.JSON(http.StatusInternalServerError, &api.ErrorModel{
			Type:     baseURL,
			Title:    "Internal Server Error",
			Status:   http.StatusInternalServerError,
			Detail:   "An error occurred while updating the agent group.",
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
	baseURL := ginutil.GetErrorTypeURI(ctx)
	name := ctx.Param("name")

	err := c.agentGroupUsecase.DeleteAgentGroup(ctx.Request.Context(), name)
	if err != nil {
		if errors.Is(err, domainport.ErrResourceNotExist) {
			ctx.JSON(http.StatusNotFound, &api.ErrorModel{
				Type:     baseURL,
				Title:    "Not Found",
				Status:   http.StatusNotFound,
				Detail:   "The agent group to delete does not exist.",
				Instance: ctx.Request.URL.String(),
				Errors: []*api.ErrorDetail{
					{
						Message:  "agent group not found",
						Location: "path.name",
						Value:    name,
					},
				},
			})

			return
		}

		c.logger.Error("failed to delete agent group", slog.String("error", err.Error()))
		ctx.JSON(http.StatusInternalServerError, &api.ErrorModel{
			Type:     baseURL,
			Title:    "Internal Server Error",
			Status:   http.StatusInternalServerError,
			Detail:   "An error occurred while deleting the agent group.",
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

	ctx.Status(http.StatusNoContent)
}
