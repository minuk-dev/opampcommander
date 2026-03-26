// Package rbac provides HTTP handlers for managing RBAC policies.
package rbac

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// unassignRoleRequest represents a request to unassign a role from a user.
type unassignRoleRequest struct {
	// UserID is the ID of the user to unassign the role from.
	UserID string `json:"userID"`
	// RoleID is the ID of the role to unassign.
	RoleID string `json:"roleID"`
}

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
			Path:        "/api/v1/rbac/assign",
			Handler:     "http.v1.rbac.AssignRole",
			HandlerFunc: c.AssignRole,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/rbac/unassign",
			Handler:     "http.v1.rbac.UnassignRole",
			HandlerFunc: c.UnassignRole,
		},
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

// AssignRole assigns a role to a user.
//
// @Summary  Assign Role
// @Tags rbac
// @Description Assign a role to a user.
// @Accept json
// @Produce json
// @Param request body v1.AssignRoleRequest true "Role assignment request"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/rbac/assign [post].
func (c *Controller) AssignRole(ctx *gin.Context) {
	var req v1.AssignRoleRequest

	err := ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	err = c.rbacUsecase.AssignRole(ctx.Request.Context(), &req)
	if err != nil {
		c.logger.Error("failed to assign role", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while assigning the role.")

		return
	}

	ctx.Status(http.StatusNoContent)
}

// UnassignRole unassigns a role from a user.
//
// @Summary  Unassign Role
// @Tags rbac
// @Description Unassign a role from a user.
// @Accept json
// @Produce json
// @Param request body unassignRoleRequest true "Role unassignment request"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/rbac/unassign [post].
func (c *Controller) UnassignRole(ctx *gin.Context) {
	var req unassignRoleRequest

	err := ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		ginutil.HandleValidationError(ctx, "userID", req.UserID, err, false)

		return
	}

	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		ginutil.HandleValidationError(ctx, "roleID", req.RoleID, err, false)

		return
	}

	err = c.rbacUsecase.UnassignRole(ctx.Request.Context(), userID, roleID)
	if err != nil {
		c.logger.Error("failed to unassign role", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while unassigning the role.")

		return
	}

	ctx.Status(http.StatusNoContent)
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
