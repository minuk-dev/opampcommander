// Package agentremoteconfig contains controller for agent remote config endpoints.
package agentremoteconfig

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

// agentRemoteConfigByNamePath is the per-item route, repeated by the read, update and delete
// entries below so a rename happens in one place.
const agentRemoteConfigByNamePath = "/api/v1/namespaces/:namespace/agentremoteconfigs/:name"

// Controller is a struct that implements the agent remote config controller.
type Controller struct {
	logger *slog.Logger

	agentRemoteConfigUsecase usecase.AgentRemoteConfigManageUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase usecase.AgentRemoteConfigManageUsecase,
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
			Path:        agentRemoteConfigByNamePath,
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
			Path:        agentRemoteConfigByNamePath,
			Handler:     "http.v1.agentremoteconfig.Update",
			HandlerFunc: c.Update,
		},
		{
			Method:      http.MethodPatch,
			Path:        agentRemoteConfigByNamePath,
			Handler:     "http.v1.agentremoteconfig.Patch",
			HandlerFunc: c.Patch,
		},
		{
			Method:      http.MethodDelete,
			Path:        agentRemoteConfigByNamePath,
			Handler:     "http.v1.agentremoteconfig.Delete",
			HandlerFunc: c.Delete,
		},
	}
}

// List retrieves a list of agent remote configs.
//
// @Summary  List Agent Remote Configs
// @Tags agentremoteconfig
// @Description Retrieve a list of agent remote configs in a namespace.
// @Success 200 {object} v1.ListResponse[v1.AgentRemoteConfig]
// @Param namespace path string true "Namespace"
// @Param limit query int false "Maximum number of agent remote configs to return"
// @Param continue query string false "Token to continue listing agent remote configs"
// @Param includeDeleted query bool false "Include soft-deleted agent remote configs"
// @Param labelSelector query string false "Label selector, e.g. env=prod,tier notin (canary,dev)"
// @Param fieldSelector query string false "Field selector over the supported fields: metadata.namespace"
// @Param name query string false "Case-sensitive name prefix filter"
// @Param nameContains query string false "Case-insensitive name substring filter (scan; pass name= to bound it)"
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/agentremoteconfigs [get].
func (c *Controller) List(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "namespace", ctx.Param("namespace"), err, true)

		return
	}

	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "limit", ctx.Query("limit"), err, false,
		)

		return
	}

	continueToken := ctx.Query("continue")

	includeDeleted, err := ginutil.ParseBool(ctx, "includeDeleted", false)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "includeDeleted", ctx.Query("includeDeleted"), err, false,
		)

		return
	}

	selectors, ok := ginutil.ParseSelectors(ctx, ginutil.LabelMetadataSelector, port.AgentRemoteConfigSelectableFields)
	if !ok {
		return
	}

	response, err := c.agentRemoteConfigUsecase.ListAgentRemoteConfigs(
		ctx.Request.Context(),
		namespace, &port.ListOptions{
			LabelSelector:  selectors.Metadata,
			FieldSelector:  selectors.Field,
			NamePrefix:     selectors.NamePrefix,
			NameContains:   selectors.NameContains,
			Limit:          limit,
			Continue:       continueToken,
			IncludeDeleted: includeDeleted,
		},
	)
	if err != nil {
		c.logger.Error(
			"failed to list agent remote configs", "error", err.Error(),
		)
		ginutil.HandleDomainError(
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

	includeDeleted, err := ginutil.ParseBool(ctx, "includeDeleted", false)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "includeDeleted", ctx.Query("includeDeleted"), err, false,
		)

		return
	}

	config, err := c.agentRemoteConfigUsecase.GetAgentRemoteConfig(
		ctx.Request.Context(), namespace, name, &port.GetOptions{
			IncludeDeleted: includeDeleted,
		},
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
		ginutil.HandleDomainError(
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

	if !ginutil.RequireResourceVersion(ctx, req.Metadata.ResourceVersion) {
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

	resourceVersion, ok := ginutil.ParseResourceVersion(ctx)
	if !ok {
		return
	}

	err = c.agentRemoteConfigUsecase.DeleteAgentRemoteConfig(
		ctx.Request.Context(), namespace, name, resourceVersion,
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

// Patch partially updates an existing AgentRemoteConfig.
//
// @Summary Patch AgentRemoteConfig
// @Tags agentremoteconfig
// @Description JSON Merge Patch with optional metadata.resourceVersion. Omitted revisions allow same-field overwrite.
// @Accept application/merge-patch+json
// @Produce json
// @Param namespace path string true "Namespace"
// @Param name path string true "Resource name"
// @Param patch body object true "Merge patch with optional metadata.resourceVersion"
// @Success 200 {object} v1.AgentRemoteConfig
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 409 {object} map[string]any
// @Failure 415 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/agentremoteconfigs/{name} [patch].
func (c *Controller) Patch(ctx *gin.Context) {
	patch, ok := ginutil.ReadMergePatch(ctx)
	if !ok {
		return
	}

	result, err := c.agentRemoteConfigUsecase.PatchAgentRemoteConfig(
		ctx.Request.Context(), ctx.Param("namespace"), ctx.Param("name"), patch,
	)
	if err != nil {
		ginutil.HandleDomainError(ctx, err, "Failed to patch resource.")

		return
	}

	ctx.JSON(http.StatusOK, result)
}
