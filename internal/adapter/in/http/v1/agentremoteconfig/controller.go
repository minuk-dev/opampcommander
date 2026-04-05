// Package agentremoteconfig contains controller for agent remote config endpoints.
package agentremoteconfig

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that implements the agent remote config controller.
type Controller struct {
	logger *slog.Logger

	agentRemoteConfigUsecase port.AgentRemoteConfigManageUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase port.AgentRemoteConfigManageUsecase,
	logger *slog.Logger,
) *Controller {
	controller := &Controller{
		logger:                   logger,
		agentRemoteConfigUsecase: usecase,
	}

	return controller
}

// RoutesInfo returns the routes information for the agent remote config controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace/agentremoteconfigs",
			Handler:     "http.v1.agentremoteconfig.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace/agentremoteconfigs/:name",
			Handler:     "http.v1.agentremoteconfig.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/namespaces/:namespace/agentremoteconfigs",
			Handler:     "http.v1.agentremoteconfig.Create",
			HandlerFunc: c.Create,
		},
		{
			Method:      http.MethodPut,
			Path:        "/api/v1/namespaces/:namespace/agentremoteconfigs/:name",
			Handler:     "http.v1.agentremoteconfig.Update",
			HandlerFunc: c.Update,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/api/v1/namespaces/:namespace/agentremoteconfigs/:name",
			Handler:     "http.v1.agentremoteconfig.Delete",
			HandlerFunc: c.Delete,
		},
	}
}

// List retrieves a list of agent remote configs.
func (c *Controller) List(ctx *gin.Context) {
	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "limit", ctx.Query("limit"), err, false,
		)

		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.agentRemoteConfigUsecase.ListAgentRemoteConfigs(
		ctx.Request.Context(), &model.ListOptions{
			Limit:          limit,
			Continue:       continueToken,
			IncludeDeleted: false,
		},
	)
	if err != nil {
		c.logger.Error(
			"failed to list agent remote configs", "error", err.Error(),
		)
		ginutil.InternalServerError(
			ctx, err,
			"An error occurred while retrieving agent remote configs.",
		)

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves an agent remote config by its name.
func (c *Controller) Get(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "namespace", ctx.Param("namespace"), err, true,
		)

		return
	}

	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "name", ctx.Param("name"), err, true,
		)

		return
	}

	config, err := c.agentRemoteConfigUsecase.GetAgentRemoteConfig(
		ctx.Request.Context(), namespace, name,
	)
	if err != nil {
		c.logger.Error(
			"failed to get agent remote config",
			"name", name, "error", err.Error(),
		)
		ginutil.HandleDomainError(
			ctx, err,
			"An error occurred while retrieving the agent remote config.",
		)

		return
	}

	ctx.JSON(http.StatusOK, config)
}

// Create creates a new agent remote config.
func (c *Controller) Create(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "namespace", ctx.Param("namespace"), err, true,
		)

		return
	}

	var req v1.AgentRemoteConfig

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	req.Metadata.Namespace = namespace

	created, err := c.agentRemoteConfigUsecase.CreateAgentRemoteConfig(
		ctx.Request.Context(), &req,
	)
	if err != nil {
		c.logger.Error(
			"failed to create agent remote config", "error", err.Error(),
		)
		ginutil.InternalServerError(
			ctx, err,
			"An error occurred while creating the agent remote config.",
		)

		return
	}

	ctx.Header(
		"Location",
		"/api/v1/namespaces/"+namespace+
			"/agentremoteconfigs/"+created.Metadata.Name,
	)
	ctx.JSON(http.StatusCreated, created)
}

// Update updates an existing agent remote config.
func (c *Controller) Update(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "namespace", ctx.Param("namespace"), err, true,
		)

		return
	}

	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "name", ctx.Param("name"), err, true,
		)

		return
	}

	var req v1.AgentRemoteConfig

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	updated, err := c.agentRemoteConfigUsecase.UpdateAgentRemoteConfig(
		ctx.Request.Context(), namespace, name, &req,
	)
	if err != nil {
		c.logger.Error(
			"failed to update agent remote config",
			"name", name, "error", err.Error(),
		)
		ginutil.HandleDomainError(
			ctx, err,
			"An error occurred while updating the agent remote config.",
		)

		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// Delete deletes an agent remote config by its name.
func (c *Controller) Delete(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "namespace", ctx.Param("namespace"), err, true,
		)

		return
	}

	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "name", ctx.Param("name"), err, true,
		)

		return
	}

	err = c.agentRemoteConfigUsecase.DeleteAgentRemoteConfig(
		ctx.Request.Context(), namespace, name,
	)
	if err != nil {
		c.logger.Error(
			"failed to delete agent remote config",
			"name", name, "error", err.Error(),
		)
		ginutil.HandleDomainError(
			ctx, err,
			"An error occurred while deleting the agent remote config.",
		)

		return
	}

	ctx.Status(http.StatusNoContent)
}
