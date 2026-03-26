// Package user provides HTTP handlers for managing users.
package user

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that implements the user controller.
type Controller struct {
	logger *slog.Logger

	// usecases
	userUsecase Usecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase Usecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		logger:      logger,
		userUsecase: usecase,
	}
}

// RoutesInfo returns the routes information for the user controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/users/me",
			Handler:     "http.v1.user.Me",
			HandlerFunc: c.Me,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/users",
			Handler:     "http.v1.user.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/users/:id",
			Handler:     "http.v1.user.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/users",
			Handler:     "http.v1.user.Create",
			HandlerFunc: c.Create,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/api/v1/users/:id",
			Handler:     "http.v1.user.Delete",
			HandlerFunc: c.Delete,
		},
	}
}

// List retrieves a list of users.
//
// @Summary  List Users
// @Tags user
// @Description Retrieve a list of users.
// @Accept json
// @Produce json
// @Success 200 {object} v1.ListResponse[v1.User]
// @Param limit query int false "Maximum number of users to return"
// @Param continue query string false "Token to continue listing users"
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/users [get].
func (c *Controller) List(ctx *gin.Context) {
	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.userUsecase.ListUsers(ctx.Request.Context(), &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		c.logger.Error("failed to list users", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the list of users.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves a user by its UID.
//
// @Summary  Get User
// @Tags user
// @Description Retrieve a user by its UID.
// @Accept  json
// @Produce  json
// @Param  id path string true "UID of the user"
// @Success  200 {object} User
// @Failure  400 {object} ErrorModel
// @Failure  404 {object} ErrorModel
// @Failure  500 {object} ErrorModel
// @Router  /api/v1/users/{id} [get].
func (c *Controller) Get(ctx *gin.Context) {
	uid, err := ginutil.ParseUUID(ctx, "id")
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	user, err := c.userUsecase.GetUser(ctx.Request.Context(), uid)
	if err != nil {
		c.logger.Error("failed to get user", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the user.")

		return
	}

	ctx.JSON(http.StatusOK, user)
}

// Create creates a new user.
//
// @Summary  Create User
// @Tags user
// @Description Create a new user.
// @Accept json
// @Produce json
// @Param user body v1.User true "User to create"
// @Success 201 {object} v1.User
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/users [post].
func (c *Controller) Create(ctx *gin.Context) {
	var req v1.User

	err := ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	created, err := c.userUsecase.CreateUser(ctx.Request.Context(), &req)
	if err != nil {
		c.logger.Error("failed to create user", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while creating the user.")

		return
	}

	ctx.JSON(http.StatusCreated, created)
}

// Delete deletes a user by its UID.
//
// @Summary  Delete User
// @Tags user
// @Description Delete a user by its UID.
// @Param  id path string true "UID of the user"
// @Success  204 "No Content"
// @Failure  400 {object} ErrorModel
// @Failure  404 {object} ErrorModel
// @Failure  500 {object} ErrorModel
// @Router  /api/v1/users/{id} [delete].
func (c *Controller) Delete(ctx *gin.Context) {
	uid, err := ginutil.ParseUUID(ctx, "id")
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	err = c.userUsecase.DeleteUser(ctx.Request.Context(), uid)
	if err != nil {
		c.logger.Error("failed to delete user", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while deleting the user.")

		return
	}

	ctx.Status(http.StatusNoContent)
}

// Me retrieves the current user's profile with roles and permissions.
//
// @Summary  Get Current User Profile
// @Tags user
// @Description Retrieve the current authenticated user's profile with roles and permissions.
// @Accept json
// @Produce json
// @Success 200 {object} v1.UserProfileResponse
// @Failure 401 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/users/me [get].
func (c *Controller) Me(ctx *gin.Context) {
	secUser, err := security.GetUser(ctx)
	if err != nil || secUser == nil || !secUser.Authenticated || secUser.Email == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})

		return
	}

	profile, err := c.userUsecase.GetUserProfile(ctx.Request.Context(), *secUser.Email)
	if err != nil {
		c.logger.Error("failed to get user profile", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the user profile.")

		return
	}

	ctx.JSON(http.StatusOK, profile)
}
