// Package role provides HTTP handlers for managing roles.
package role

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that implements the role controller.
type Controller struct {
	logger *slog.Logger

	// usecases
	roleUsecase Usecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase Usecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		logger:      logger,
		roleUsecase: usecase,
	}
}

// RoutesInfo returns the routes information for the role controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/roles",
			Handler:     "http.v1.role.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/roles/:id",
			Handler:     "http.v1.role.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/roles",
			Handler:     "http.v1.role.Create",
			HandlerFunc: c.Create,
		},
		{
			Method:      http.MethodPut,
			Path:        "/api/v1/roles/:id",
			Handler:     "http.v1.role.Update",
			HandlerFunc: c.Update,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/api/v1/roles/:id",
			Handler:     "http.v1.role.Delete",
			HandlerFunc: c.Delete,
		},
	}
}

// List retrieves a list of roles.
//
// @Summary  List Roles
// @Tags role
// @Description Retrieve a list of roles.
// @Accept json
// @Produce json
// @Success 200 {object} v1.ListResponse[v1.Role]
// @Param limit query int false "Maximum number of roles to return"
// @Param continue query string false "Token to continue listing roles"
// @Param includeDeleted query bool false "Include soft-deleted roles"
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/roles [get].
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

	response, err := c.roleUsecase.ListRoles(ctx.Request.Context(), &model.ListOptions{
		Limit:          limit,
		Continue:       continueToken,
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		c.logger.Error("failed to list roles", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the list of roles.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves a role by its UID.
//
// @Summary  Get Role
// @Tags role
// @Description Retrieve a role by its UID.
// @Accept  json
// @Produce  json
// @Param  id path string true "UID of the role"
// @Param  includeDeleted query bool false "Include soft-deleted role"
// @Success  200 {object} Role
// @Failure  400 {object} ErrorModel
// @Failure  404 {object} ErrorModel
// @Failure  500 {object} ErrorModel
// @Router  /api/v1/roles/{id} [get].
func (c *Controller) Get(ctx *gin.Context) {
	uid, err := ginutil.ParseUUID(ctx, "id")
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	includeDeleted, err := ginutil.ParseBool(ctx, "includeDeleted", false)
	if err != nil {
		ginutil.HandleValidationError(ctx, "includeDeleted", ctx.Query("includeDeleted"), err, false)

		return
	}

	role, err := c.roleUsecase.GetRole(ctx.Request.Context(), uid, &model.GetOptions{
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		c.logger.Error("failed to get role", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the role.")

		return
	}

	ctx.JSON(http.StatusOK, role)
}

// Create creates a new role.
//
// @Summary  Create Role
// @Tags role
// @Description Create a new role.
// @Accept json
// @Produce json
// @Param role body v1.Role true "Role to create"
// @Success 201 {object} v1.Role
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/roles [post].
func (c *Controller) Create(ctx *gin.Context) {
	var req v1.Role

	err := ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	created, err := c.roleUsecase.CreateRole(ctx.Request.Context(), &req)
	if err != nil {
		c.logger.Error("failed to create role", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while creating the role.")

		return
	}

	ctx.JSON(http.StatusCreated, created)
}

// Update updates an existing role.
//
// @Summary  Update Role
// @Tags role
// @Description Update an existing role.
// @Accept json
// @Produce json
// @Param  id path string true "UID of the role"
// @Param role body v1.Role true "Updated role"
// @Success 200 {object} v1.Role
// @Failure 400 {object} ErrorModel
// @Failure 404 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/roles/{id} [put].
func (c *Controller) Update(ctx *gin.Context) {
	uid, err := ginutil.ParseUUID(ctx, "id")
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	var req v1.Role

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	updated, err := c.roleUsecase.UpdateRole(ctx.Request.Context(), uid, &req)
	if err != nil {
		c.logger.Error("failed to update role", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while updating the role.")

		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// Delete deletes a role by its UID.
//
// @Summary  Delete Role
// @Tags role
// @Description Delete a role by its UID.
// @Param  id path string true "UID of the role"
// @Success  204 "No Content"
// @Failure  400 {object} ErrorModel
// @Failure  404 {object} ErrorModel
// @Failure  500 {object} ErrorModel
// @Router  /api/v1/roles/{id} [delete].
func (c *Controller) Delete(ctx *gin.Context) {
	uid, err := ginutil.ParseUUID(ctx, "id")
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	err = c.roleUsecase.DeleteRole(ctx.Request.Context(), uid)
	if err != nil {
		c.logger.Error("failed to delete role", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while deleting the role.")

		return
	}

	ctx.Status(http.StatusNoContent)
}
