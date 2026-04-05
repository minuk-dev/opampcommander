// Package namespace contains controller for namespace related endpoints.
package namespace

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that implements the namespace controller.
type Controller struct {
	logger           *slog.Logger
	namespaceUsecase port.NamespaceManageUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase port.NamespaceManageUsecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		logger:           logger,
		namespaceUsecase: usecase,
	}
}

// RoutesInfo returns the routes information for the namespace controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces",
			Handler:     "http.v1.namespace.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace",
			Handler:     "http.v1.namespace.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/namespaces",
			Handler:     "http.v1.namespace.Create",
			HandlerFunc: c.Create,
		},
		{
			Method:      http.MethodPut,
			Path:        "/api/v1/namespaces/:namespace",
			Handler:     "http.v1.namespace.Update",
			HandlerFunc: c.Update,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/api/v1/namespaces/:namespace",
			Handler:     "http.v1.namespace.Delete",
			HandlerFunc: c.Delete,
		},
	}
}

// List retrieves a list of namespaces.
func (c *Controller) List(ctx *gin.Context) {
	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "limit", ctx.Query("limit"), err, false,
		)

		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.namespaceUsecase.ListNamespaces(
		ctx.Request.Context(),
		&model.ListOptions{
			Limit:          limit,
			Continue:       continueToken,
			IncludeDeleted: false,
		},
	)
	if err != nil {
		c.logger.Error(
			"failed to list namespaces",
			"error", err.Error(),
		)
		ginutil.InternalServerError(
			ctx, err,
			"An error occurred while retrieving namespaces.",
		)

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves a namespace by name.
func (c *Controller) Get(ctx *gin.Context) {
	name, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "namespace", ctx.Param("namespace"), err, true,
		)

		return
	}

	namespace, err := c.namespaceUsecase.GetNamespace(
		ctx.Request.Context(), name,
	)
	if err != nil {
		c.logger.Error(
			"failed to get namespace",
			"name", name, "error", err.Error(),
		)
		ginutil.HandleDomainError(
			ctx, err,
			"An error occurred while retrieving the namespace.",
		)

		return
	}

	ctx.JSON(http.StatusOK, namespace)
}

// Create creates a new namespace.
func (c *Controller) Create(ctx *gin.Context) {
	var req v1.Namespace

	err := ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "body", "", err, false,
		)

		return
	}

	created, err := c.namespaceUsecase.CreateNamespace(
		ctx.Request.Context(), &req,
	)
	if err != nil {
		c.logger.Error(
			"failed to create namespace",
			"error", err.Error(),
		)
		ginutil.InternalServerError(
			ctx, err,
			"An error occurred while creating the namespace.",
		)

		return
	}

	ctx.Header(
		"Location",
		"/api/v1/namespaces/"+created.Metadata.Name,
	)
	ctx.JSON(http.StatusCreated, created)
}

// Update updates an existing namespace.
func (c *Controller) Update(ctx *gin.Context) {
	name, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "namespace", ctx.Param("namespace"), err, true,
		)

		return
	}

	var req v1.Namespace

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "body", "", err, false,
		)

		return
	}

	updated, err := c.namespaceUsecase.UpdateNamespace(
		ctx.Request.Context(), name, &req,
	)
	if err != nil {
		c.logger.Error(
			"failed to update namespace",
			"name", name, "error", err.Error(),
		)
		ginutil.HandleDomainError(
			ctx, err,
			"An error occurred while updating the namespace.",
		)

		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// Delete deletes a namespace by name.
func (c *Controller) Delete(ctx *gin.Context) {
	name, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(
			ctx, "namespace", ctx.Param("namespace"), err, true,
		)

		return
	}

	err = c.namespaceUsecase.DeleteNamespace(
		ctx.Request.Context(), name,
	)
	if err != nil {
		c.logger.Error(
			"failed to delete namespace",
			"name", name, "error", err.Error(),
		)
		ginutil.HandleDomainError(
			ctx, err,
			"An error occurred while deleting the namespace.",
		)

		return
	}

	ctx.Status(http.StatusNoContent)
}
