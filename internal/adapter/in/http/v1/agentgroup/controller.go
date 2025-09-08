// Package agentgroup provides HTTP handlers for managing agent groups.
package agentgroup

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

type AgentGroupUsecase = domainport.AgentGroupUsecase

type Controller struct {
	logger *slog.Logger

	// usecases
	agentGroupUsecase AgentGroupUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase AgentGroupUsecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
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
			Path:        "/api/v1/agentgroups/:id",
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
			Path:        "/api/v1/agentgroups/:id",
			Handler:     "http.v1.agentgroup.Update",
			HandlerFunc: c.Update,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/api/v1/agentgroups/:id",
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
}

// Get retrieves an agent group by its ID.
//
// @Summary Get Agent Group
// @Tags agentgroup
// @Description Retrieve an agent group by its ID.
// @Accept json
// @Produce json
// @Success 200 {object} AgentGroup
// @Param id path string true "Agent Group ID"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentgroups/{id} [get].
func (c *Controller) Get(ctx *gin.Context) {
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
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentgroups [post].
func (c *Controller) Create(ctx *gin.Context) {
}

// Update updates an existing agent group.
//
// @Summary Update Agent Group
// @Tags agentgroup
// @Description Update an existing agent group.
// @Accept json
// @Produce json
// @Param id path string true "Agent Group ID"
// @Param agentGroup body AgentGroup true "Updated Agent Group"
// @Success 200 {object} AgentGroup
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentgroups/{id} [put].
func (c *Controller) Update(ctx *gin.Context) {
}

// Delete marks an agent group as deleted.
//
// @Summary Delete Agent Group
// @Tags agentgroup
// @Description Mark an agent group as deleted.
// @Param id path string true "Agent Group ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentgroups/{id} [delete].
func (c *Controller) Delete(ctx *gin.Context) {
}
