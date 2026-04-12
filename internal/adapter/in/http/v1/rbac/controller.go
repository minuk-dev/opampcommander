// Package rbac provides HTTP handlers for managing RBAC policies.
package rbac

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that implements the RBAC controller.
type Controller struct {
	logger *slog.Logger

	// usecases
	rbacUsecase Usecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase Usecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		logger:      logger,
		rbacUsecase: usecase,
	}
}

// RoutesInfo returns the routes information for the RBAC controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/rbac/check",
			Handler:     "http.v1.rbac.CheckPermission",
			HandlerFunc: c.CheckPermission,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/rbac/users/:id/roles",
			Handler:     "http.v1.rbac.GetUserRoles",
			HandlerFunc: c.GetUserRoles,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/rbac/users/:id/permissions",
			Handler:     "http.v1.rbac.GetUserPermissions",
			HandlerFunc: c.GetUserPermissions,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/rbac/sync",
			Handler:     "http.v1.rbac.SyncPolicies",
			HandlerFunc: c.SyncPolicies,
		},
	}
}

// CheckPermission checks whether a user has a specific permission.
//
// @Summary  Check Permission
// @Tags rbac
// @Description Check whether a user has a specific permission.
// @Accept json
// @Produce json
// @Param request body v1.CheckPermissionRequest true "Permission check request"
// @Success 200 {object} v1.CheckPermissionResponse
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/rbac/check [post].
func (c *Controller) CheckPermission(ctx *gin.Context) {
	var req v1.CheckPermissionRequest

	err := ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	response, err := c.rbacUsecase.CheckPermission(ctx.Request.Context(), &req)
	if err != nil {
		c.logger.Error("failed to check permission", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while checking the permission.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetUserRoles retrieves the roles assigned to a user.
//
// @Summary  Get User Roles
// @Tags rbac
// @Description Retrieve the roles assigned to a user.
// @Accept  json
// @Produce  json
// @Param  id path string true "UID of the user"
// @Success  200 {object} v1.ListResponse[v1.Role]
// @Failure  400 {object} ErrorModel
// @Failure  404 {object} ErrorModel
// @Failure  500 {object} ErrorModel
// @Router  /api/v1/rbac/users/{id}/roles [get].
func (c *Controller) GetUserRoles(ctx *gin.Context) {
	userID, err := ginutil.ParseUUID(ctx, "id")
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	response, err := c.rbacUsecase.GetUserRoles(ctx.Request.Context(), userID)
	if err != nil {
		c.logger.Error("failed to get user roles", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the user roles.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetUserPermissions retrieves the permissions of a user.
//
// @Summary  Get User Permissions
// @Tags rbac
// @Description Retrieve the permissions of a user.
// @Accept  json
// @Produce  json
// @Param  id path string true "UID of the user"
// @Success  200 {object} v1.ListResponse[v1.Permission]
// @Failure  400 {object} ErrorModel
// @Failure  404 {object} ErrorModel
// @Failure  500 {object} ErrorModel
// @Router  /api/v1/rbac/users/{id}/permissions [get].
func (c *Controller) GetUserPermissions(ctx *gin.Context) {
	userID, err := ginutil.ParseUUID(ctx, "id")
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	response, err := c.rbacUsecase.GetUserPermissions(ctx.Request.Context(), userID)
	if err != nil {
		c.logger.Error("failed to get user permissions", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the user permissions.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// SyncPolicies synchronizes RBAC policies.
//
// @Summary  Sync Policies
// @Tags rbac
// @Description Synchronize RBAC policies.
// @Accept json
// @Produce json
// @Success 204 "No Content"
// @Failure 500 {object} ErrorModel
// @Router /api/v1/rbac/sync [post].
func (c *Controller) SyncPolicies(ctx *gin.Context) {
	err := c.rbacUsecase.SyncPolicies(ctx.Request.Context())
	if err != nil {
		c.logger.Error("failed to sync policies", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while synchronizing RBAC policies.")

		return
	}

	ctx.Status(http.StatusNoContent)
}
