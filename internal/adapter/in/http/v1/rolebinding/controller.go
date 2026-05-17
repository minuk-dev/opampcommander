// Package rolebinding provides HTTP handlers for managing role bindings.
package rolebinding

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that implements the role binding controller.
type Controller struct {
	logger  *slog.Logger
	usecase Usecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase Usecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		logger:  logger,
		usecase: usecase,
	}
}

// RoutesInfo returns the routes information for the role binding controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace/rolebindings",
			Handler:     "http.v1.rolebinding.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace/rolebindings/:name",
			Handler:     "http.v1.rolebinding.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/namespaces/:namespace/rolebindings",
			Handler:     "http.v1.rolebinding.Create",
			HandlerFunc: c.Create,
		},
		{
			Method:      http.MethodPut,
			Path:        "/api/v1/namespaces/:namespace/rolebindings/:name",
			Handler:     "http.v1.rolebinding.Update",
			HandlerFunc: c.Update,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/api/v1/namespaces/:namespace/rolebindings/:name",
			Handler:     "http.v1.rolebinding.Delete",
			HandlerFunc: c.Delete,
		},
	}
}

// List retrieves a list of role bindings.
//
// @Summary List RoleBindings
// @Tags rolebinding
// @Description Retrieves a list of role bindings with pagination options.
// @Success 200 {object} v1.ListResponse[v1.RoleBinding]
// @Param limit query int false "Maximum number of role bindings to return"
// @Param continue query string false "Token to continue listing"
// @Param includeDeleted query bool false "Include soft-deleted role bindings"
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/rolebindings [get].
func (c *Controller) List(ctx *gin.Context) {
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

	response, err := c.usecase.ListRoleBindings(ctx.Request.Context(), &model.ListOptions{
		Limit:          limit,
		Continue:       continueToken,
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		c.logger.Error("failed to list role bindings", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while retrieving the list of role bindings.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves a role binding by namespace and name.
//
// @Summary Get RoleBinding
// @Tags rolebinding
// @Description Retrieve a role binding by namespace and name.
// @Accept json
// @Produce json
// @Success 200 {object} v1.RoleBinding
// @Param namespace path string true "Namespace"
// @Param name path string true "RoleBinding Name"
// @Param includeDeleted query bool false "Include soft-deleted role binding"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/rolebindings/{name} [get].
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

	roleBinding, err := c.usecase.GetRoleBinding(ctx.Request.Context(), namespace, name, &model.GetOptions{
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		c.logger.Error("failed to get role binding", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the role binding.")

		return
	}

	ctx.JSON(http.StatusOK, roleBinding)
}

// Create creates a new role binding.
//
// @Summary Create RoleBinding
// @Tags rolebinding
// @Description Create a new role binding.
// @Accept json
// @Produce json
// @Param namespace path string true "Namespace"
// @Param roleBinding body v1.RoleBinding true "RoleBinding to create"
// @Success 201 {object} v1.RoleBinding
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/rolebindings [post].
func (c *Controller) Create(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "namespace", ctx.Param("namespace"), err, true)

		return
	}

	var req v1.RoleBinding

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	req.Metadata.Namespace = namespace

	created, err := c.usecase.CreateRoleBinding(ctx.Request.Context(), &req)
	if err != nil {
		c.logger.Error("failed to create role binding", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while creating the role binding.")

		return
	}

	ctx.Header("Location", "/api/v1/namespaces/"+namespace+"/rolebindings/"+created.Metadata.Name)
	ctx.JSON(http.StatusCreated, created)
}

// Update updates an existing role binding.
//
// @Summary Update RoleBinding
// @Tags rolebinding
// @Description Update an existing role binding.
// @Accept json
// @Produce json
// @Param namespace path string true "Namespace"
// @Param name path string true "RoleBinding Name"
// @Param roleBinding body v1.RoleBinding true "Updated RoleBinding"
// @Success 200 {object} v1.RoleBinding
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/rolebindings/{name} [put].
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

	var req v1.RoleBinding

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	updated, err := c.usecase.UpdateRoleBinding(ctx.Request.Context(), namespace, name, &req)
	if err != nil {
		c.logger.Error("failed to update role binding", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while updating the role binding.")

		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// Delete marks a role binding as deleted.
//
// @Summary Delete RoleBinding
// @Tags rolebinding
// @Description Mark a role binding as deleted.
// @Param namespace path string true "Namespace"
// @Param name path string true "RoleBinding Name"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/rolebindings/{name} [delete].
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

	err = c.usecase.DeleteRoleBinding(ctx.Request.Context(), namespace, name)
	if err != nil {
		c.logger.Error("failed to delete role binding", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while deleting the role binding.")

		return
	}

	ctx.Status(http.StatusNoContent)
}
