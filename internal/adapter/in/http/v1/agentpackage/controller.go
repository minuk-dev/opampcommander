// Package agentpackage contains controller for agent package related endpoints.
package agentpackage

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that implements the agent package controller.
type Controller struct {
	logger *slog.Logger

	agentpackageUsecase port.AgentPackageManageUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase port.AgentPackageManageUsecase,
	logger *slog.Logger,
) *Controller {
	controller := &Controller{
		logger:              logger,
		agentpackageUsecase: usecase,
	}

	return controller
}

// RoutesInfo returns the routes information for the agent package controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/agentpackages",
			Handler:     "http.v1.agentpackage.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/agentpackages/:name",
			Handler:     "http.v1.agentpackage.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/agentpackages",
			Handler:     "http.v1.agentpackage.Create",
			HandlerFunc: c.Create,
		},
		{
			Method:      http.MethodPut,
			Path:        "/api/v1/agentpackages/:name",
			Handler:     "http.v1.agentpackage.Update",
			HandlerFunc: c.Update,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/api/v1/agentpackages/:name",
			Handler:     "http.v1.agentpackage.Delete",
			HandlerFunc: c.Delete,
		},
	}
}

// List retrieves a list of agent packages.
//
// @Summary  List Agent Packages
// @Tags agentpackage
// @Description Retrieve a list of agent packages.
// @Success 200 {object} v1.ListResponse[v1.AgentPackage]
// @Param limit query int false "Maximum number of agent packages to return"
// @Param continue query string false "Token to continue listing agent packages"
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentpackages [get].
func (c *Controller) List(ctx *gin.Context) {
	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.agentpackageUsecase.ListAgentPackages(ctx.Request.Context(), &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		c.logger.Error("failed to list agent packages", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while retrieving the list of agent packages.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves an agent package by its name.
//
// @Summary  Get Agent Package
// @Tags agentpackage
// @Description Retrieve an agent package by its name.
// @Success 200 {object} v1.AgentPackage
// @Param name path string true "Name of the agent package"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentpackages/{name} [get].
func (c *Controller) Get(ctx *gin.Context) {
	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "name", ctx.Param("name"), err, true)

		return
	}

	agentPackage, err := c.agentpackageUsecase.GetAgentPackage(ctx.Request.Context(), name)
	if err != nil {
		c.logger.Error("failed to get agent package", "name", name, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the agent package.")

		return
	}

	ctx.JSON(http.StatusOK, agentPackage)
}

// Create creates a new agent package.
//
// @Summary  Create Agent Package
// @Tags agentpackage
// @Description Create a new agent package.
// @Accept json
// @Produce json
// @Success 201 {object} v1.AgentPackage
// @Param agentPackage body v1.AgentPackage true "Agent Package to create"
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentpackages [post].
func (c *Controller) Create(ctx *gin.Context) {
	var req v1.AgentPackage

	err := ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	created, err := c.agentpackageUsecase.CreateAgentPackage(ctx.Request.Context(), &req)
	if err != nil {
		c.logger.Error("failed to create agent package", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while creating the agent package.")

		return
	}

	ctx.Header("Location", "/api/v1/agentpackages/"+created.Metadata.Name)
	ctx.JSON(http.StatusCreated, created)
}

// Update updates an existing agent package.
//
// @Summary  Update Agent Package
// @Tags agentpackage
// @Description Update an existing agent package.
// @Accept json
// @Produce json
// @Success 200 {object} v1.AgentPackage
// @Param name path string true "Name of the agent package"
// @Param agentPackage body v1.AgentPackage true "Updated Agent Package"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentpackages/{name} [put].
func (c *Controller) Update(ctx *gin.Context) {
	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "name", ctx.Param("name"), err, true)

		return
	}

	var req v1.AgentPackage

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	updated, err := c.agentpackageUsecase.UpdateAgentPackage(ctx.Request.Context(), name, &req)
	if err != nil {
		c.logger.Error("failed to update agent package", "name", name, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while updating the agent package.")

		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// Delete deletes an agent package by its name.
//
// @Summary  Delete Agent Package
// @Tags agentpackage
// @Description Delete an agent package by its name.
// @Param name path string true "Name of the agent package"
// @Success 204
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/agentpackages/{name} [delete].
func (c *Controller) Delete(ctx *gin.Context) {
	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "name", ctx.Param("name"), err, true)

		return
	}

	err = c.agentpackageUsecase.DeleteAgentPackage(ctx.Request.Context(), name)
	if err != nil {
		c.logger.Error("failed to delete agent package", "name", name, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while deleting the agent package.")

		return
	}

	ctx.Status(http.StatusNoContent)
}
