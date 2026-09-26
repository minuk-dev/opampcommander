// Package application contains controller for application related endpoints.
package application

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

// Controller is a struct that implements the application controller.
type Controller struct {
	logger             *slog.Logger
	applicationUsecase ManageUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase ManageUsecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		logger:             logger,
		applicationUsecase: usecase,
	}
}

// RoutesInfo returns the routes information for the application controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/applications",
			Handler:     "http.v1.application.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/applications/:id",
			Handler:     "http.v1.application.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/applications/:id/agents",
			Handler:     "http.v1.application.ListAgents",
			HandlerFunc: c.ListAgents,
		},
	}
}

// List retrieves a list of applications.
//
// @Summary  List Applications
// @Tags application
// @Description Retrieve a list of discovered applications.
// @Accept json
// @Produce json
// @Success 200 {object} v1.ListResponse[v1.Application]
// @Param limit query int false "Maximum number of applications to return"
// @Param continue query string false "Token to continue listing applications"
// @Param labelSelector query string false "Label selector, e.g. env=prod,tier notin (canary,dev)"
// @Param fieldSelector query string false "Field selector over the supported fields: spec.platform"
// @Param name query string false "Case-sensitive name prefix filter"
// @Param nameContains query string false "Case-insensitive name substring filter (scan; pass name= to bound it)"
// @Failure 400 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/applications [get].
func (c *Controller) List(ctx *gin.Context) {
	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	selectors, ok := ginutil.ParseSelectors(ctx, ginutil.LabelMetadataSelector, applicationport.ApplicationSelectableFields)
	if !ok {
		return
	}

	var response *v1.ListResponse[v1.Application]

	response, err = c.applicationUsecase.ListApplications(
		ctx.Request.Context(),
		&applicationport.ListOptions{
			LabelSelector:  selectors.Metadata,
			FieldSelector:  selectors.Field,
			NamePrefix:     selectors.NamePrefix,
			NameContains:   selectors.NameContains,
			Limit:          limit,
			Continue:       ctx.Query("continue"),
			IncludeDeleted: false,
			ConnectedOnly:  false,
		},
	)
	if err != nil {
		c.logger.Error("failed to list applications", "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving applications.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves a application by ID.
//
// @Summary  Get Application
// @Tags application
// @Description Retrieve a discovered application by its ID.
// @Accept json
// @Produce json
// @Success 200 {object} v1.Application
// @Param id path string true "Application ID"
// @Failure 400 {object} ErrorModel
// @Failure 404 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/applications/{id} [get].
func (c *Controller) Get(ctx *gin.Context) {
	id, err := ginutil.ParseString(ctx, "id", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	var application *v1.Application

	application, err = c.applicationUsecase.GetApplication(ctx.Request.Context(), id)
	if err != nil {
		c.logger.Error("failed to get application", "id", id, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the application.")

		return
	}

	ctx.JSON(http.StatusOK, application)
}

// ListAgents retrieves the agents associated with a application.
//
// @Summary  List Application Agents
// @Tags application
// @Description Retrieve the agents running in a discovered application.
// @Accept json
// @Produce json
// @Success 200 {object} v1.ListResponse[v1.Agent]
// @Param id path string true "Application ID"
// @Param limit query int false "Maximum number of agents to return"
// @Param continue query string false "Token to continue listing agents"
// @Failure 400 {object} ErrorModel
// @Failure 404 {object} ErrorModel
// @Failure 500 {object} ErrorModel
// @Router /api/v1/applications/{id}/agents [get].
func (c *Controller) ListAgents(ctx *gin.Context) {
	id, err := ginutil.ParseString(ctx, "id", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "id", ctx.Param("id"), err, true)

		return
	}

	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	var response *v1.ListResponse[v1.Agent]

	response, err = c.applicationUsecase.ListAgentsByApplication(
		ctx.Request.Context(),
		id,
		&applicationport.ListOptions{
			Limit:          limit,
			Continue:       ctx.Query("continue"),
			IncludeDeleted: false,
			ConnectedOnly:  false,
		},
	)
	if err != nil {
		c.logger.Error("failed to list application agents", "id", id, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the application agents.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}
