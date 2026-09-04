// Package remoteconfigschema contains the controller for remote config schema endpoints.
package remoteconfigschema

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

// Controller implements the remote config schema controller.
type Controller struct {
	logger *slog.Logger

	schemaUsecase usecase.RemoteConfigSchemaManageUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase usecase.RemoteConfigSchemaManageUsecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		logger:        logger,
		schemaUsecase: usecase,
	}
}

// RoutesInfo returns the routes information for the remote config schema controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace/remoteconfigschemas",
			Handler:     "http.v1.remoteconfigschema.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace/remoteconfigschemas/:name",
			Handler:     "http.v1.remoteconfigschema.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/namespaces/:namespace/remoteconfigschemas",
			Handler:     "http.v1.remoteconfigschema.Create",
			HandlerFunc: c.Create,
		},
		{
			Method:      http.MethodPut,
			Path:        "/api/v1/namespaces/:namespace/remoteconfigschemas/:name",
			Handler:     "http.v1.remoteconfigschema.Update",
			HandlerFunc: c.Update,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/api/v1/namespaces/:namespace/remoteconfigschemas/:name",
			Handler:     "http.v1.remoteconfigschema.Delete",
			HandlerFunc: c.Delete,
		},
	}
}

// List retrieves a list of remote config schemas.
//
// @Summary  List RemoteConfigSchemas
// @Tags remoteconfigschema
// @Description Retrieve a list of remote config schemas in a namespace.
// @Success 200 {object} v1.ListResponse[v1.RemoteConfigSchema]
// @Param namespace path string true "Namespace"
// @Param limit query int false "Maximum number of schemas to return"
// @Param continue query string false "Token to continue listing schemas"
// @Param includeDeleted query bool false "Include soft-deleted schemas"
// @Param labelSelector query string false "Label selector, e.g. env=prod,tier notin (canary,dev)"
// @Param fieldSelector query string false "Fields: metadata.namespace, spec.binary, spec.version"
// @Param name query string false "Case-sensitive name prefix filter"
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/remoteconfigschemas [get].
func (c *Controller) List(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "namespace", ctx.Param("namespace"), err, true)

		return
	}

	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	continueToken := ctx.Query("continue")

	includeDeleted, err := ginutil.ParseBool(ctx, "includeDeleted", false)
	if err != nil {
		ginutil.HandleValidationError(ctx, "includeDeleted", ctx.Query("includeDeleted"), err, false)

		return
	}

	selectors, ok := ginutil.ParseSelectors(ctx, port.RemoteConfigSchemaSelectableFields)
	if !ok {
		return
	}

	response, err := c.schemaUsecase.ListRemoteConfigSchemas(
		ctx.Request.Context(), namespace, &port.ListOptions{
			LabelSelector:  selectors.Label,
			FieldSelector:  selectors.Field,
			NamePrefix:     selectors.NamePrefix,
			Limit:          limit,
			Continue:       continueToken,
			IncludeDeleted: includeDeleted,
		},
	)
	if err != nil {
		c.logger.Error("failed to list remote config schemas", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while retrieving remote config schemas.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves a remote config schema by its name.
//
// @Summary  Get RemoteConfigSchema
// @Tags remoteconfigschema
// @Description Retrieve a remote config schema by its name.
// @Success 200 {object} v1.RemoteConfigSchema
// @Param namespace path string true "Namespace"
// @Param name path string true "Name of the schema"
// @Param includeDeleted query bool false "Include soft-deleted schema"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/remoteconfigschemas/{name} [get].
func (c *Controller) Get(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "namespace", ctx.Param("namespace"), err, true)

		return
	}

	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "name", ctx.Param("name"), err, true)

		return
	}

	includeDeleted, err := ginutil.ParseBool(ctx, "includeDeleted", false)
	if err != nil {
		ginutil.HandleValidationError(ctx, "includeDeleted", ctx.Query("includeDeleted"), err, false)

		return
	}

	schema, err := c.schemaUsecase.GetRemoteConfigSchema(
		ctx.Request.Context(), namespace, name, &port.GetOptions{
			IncludeDeleted: includeDeleted,
		},
	)
	if err != nil {
		c.logger.Error("failed to get remote config schema", "name", name, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the remote config schema.")

		return
	}

	ctx.JSON(http.StatusOK, schema)
}

// Create creates a new remote config schema.
//
// @Summary  Create RemoteConfigSchema
// @Tags remoteconfigschema
// @Description Create a new remote config schema.
// @Accept json
// @Produce json
// @Success 201 {object} v1.RemoteConfigSchema
// @Param namespace path string true "Namespace"
// @Param remoteconfigschema body v1.RemoteConfigSchema true "RemoteConfigSchema to create"
// @Failure 400 {object} map[string]any
// @Failure 409 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/remoteconfigschemas [post].
func (c *Controller) Create(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "namespace", ctx.Param("namespace"), err, true)

		return
	}

	var req v1.RemoteConfigSchema

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	req.Metadata.Namespace = namespace

	created, err := c.schemaUsecase.CreateRemoteConfigSchema(ctx.Request.Context(), &req)
	if err != nil {
		c.logger.Error("failed to create remote config schema", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while creating the remote config schema.")

		return
	}

	ctx.Header(
		"Location",
		"/api/v1/namespaces/"+namespace+"/remoteconfigschemas/"+created.Metadata.Name,
	)
	ctx.JSON(http.StatusCreated, created)
}

// Update updates an existing remote config schema.
//
// @Summary  Update RemoteConfigSchema
// @Tags remoteconfigschema
// @Description Update an existing remote config schema.
// @Accept json
// @Produce json
// @Success 200 {object} v1.RemoteConfigSchema
// @Param namespace path string true "Namespace"
// @Param name path string true "Name of the schema"
// @Param remoteconfigschema body v1.RemoteConfigSchema true "Updated RemoteConfigSchema"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/remoteconfigschemas/{name} [put].
func (c *Controller) Update(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "namespace", ctx.Param("namespace"), err, true)

		return
	}

	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "name", ctx.Param("name"), err, true)

		return
	}

	var req v1.RemoteConfigSchema

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	updated, err := c.schemaUsecase.UpdateRemoteConfigSchema(ctx.Request.Context(), namespace, name, &req)
	if err != nil {
		c.logger.Error("failed to update remote config schema", "name", name, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while updating the remote config schema.")

		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// Delete deletes a remote config schema by its name.
//
// @Summary  Delete RemoteConfigSchema
// @Tags remoteconfigschema
// @Description Delete a remote config schema by its name.
// @Param namespace path string true "Namespace"
// @Param name path string true "Name of the schema"
// @Success 204
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/remoteconfigschemas/{name} [delete].
func (c *Controller) Delete(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "namespace", ctx.Param("namespace"), err, true)

		return
	}

	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "name", ctx.Param("name"), err, true)

		return
	}

	err = c.schemaUsecase.DeleteRemoteConfigSchema(ctx.Request.Context(), namespace, name)
	if err != nil {
		c.logger.Error("failed to delete remote config schema", "name", name, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while deleting the remote config schema.")

		return
	}

	ctx.Status(http.StatusNoContent)
}
