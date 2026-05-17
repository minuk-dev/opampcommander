// Package certificate contains controller for certificate related endpoints.
package certificate

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that implements the certificate controller.
type Controller struct {
	logger *slog.Logger

	certificateUsecase port.CertificateManageUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	usecase port.CertificateManageUsecase,
	logger *slog.Logger,
) *Controller {
	controller := &Controller{
		logger:             logger,
		certificateUsecase: usecase,
	}

	return controller
}

// RoutesInfo returns the routes information for the certificate controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace/certificates",
			Handler:     "http.v1.certificate.List",
			HandlerFunc: c.List,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/namespaces/:namespace/certificates/:name",
			Handler:     "http.v1.certificate.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/namespaces/:namespace/certificates",
			Handler:     "http.v1.certificate.Create",
			HandlerFunc: c.Create,
		},
		{
			Method:      http.MethodPut,
			Path:        "/api/v1/namespaces/:namespace/certificates/:name",
			Handler:     "http.v1.certificate.Update",
			HandlerFunc: c.Update,
		},
		{
			Method:      http.MethodDelete,
			Path:        "/api/v1/namespaces/:namespace/certificates/:name",
			Handler:     "http.v1.certificate.Delete",
			HandlerFunc: c.Delete,
		},
	}
}

// List retrieves a list of certificates.
//
// @Summary  List Certificates
// @Tags certificate
// @Description Retrieve a list of certificates.
// @Success 200 {object} v1.ListResponse[v1.Certificate]
// @Param namespace path string true "Namespace"
// @Param limit query int false "Maximum number of certificates to return"
// @Param continue query string false "Token to continue listing certificates"
// @Param includeDeleted query bool false "Include soft-deleted certificates"
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/certificates [get].
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

	response, err := c.certificateUsecase.ListCertificates(ctx.Request.Context(), &model.ListOptions{
		Limit:          limit,
		Continue:       continueToken,
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		c.logger.Error("failed to list certificates", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while retrieving the list of certificates.")

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Get retrieves a certificate by its name.
//
// @Summary  Get Certificate
// @Tags certificate
// @Description Retrieve a certificate by its name.
// @Success 200 {object} v1.Certificate
// @Param namespace path string true "Namespace"
// @Param name path string true "Name of the certificate"
// @Param includeDeleted query bool false "Include soft-deleted certificate"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/certificates/{name} [get].
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

	certificate, err := c.certificateUsecase.GetCertificate(ctx.Request.Context(), namespace, name, &model.GetOptions{
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		c.logger.Error("failed to get certificate", "name", name, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while retrieving the certificate.")

		return
	}

	ctx.JSON(http.StatusOK, certificate)
}

// Create creates a new certificate.
//
// @Summary  Create Certificate
// @Tags certificate
// @Description Create a new certificate.
// @Accept json
// @Produce json
// @Success 201 {object} v1.Certificate
// @Param namespace path string true "Namespace"
// @Param certificate body v1.Certificate true "Certificate to create"
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/certificates [post].
func (c *Controller) Create(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "namespace", ctx.Param("namespace"), err, true)

		return
	}

	var req v1.Certificate

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	req.Metadata.Namespace = namespace

	created, err := c.certificateUsecase.CreateCertificate(ctx.Request.Context(), &req)
	if err != nil {
		c.logger.Error("failed to create certificate", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while creating the certificate.")

		return
	}

	ctx.Header("Location", "/api/v1/namespaces/"+namespace+"/certificates/"+created.Metadata.Name)
	ctx.JSON(http.StatusCreated, created)
}

// Update updates an existing certificate.
//
// @Summary  Update Certificate
// @Tags certificate
// @Description Update an existing certificate.
// @Accept json
// @Produce json
// @Success 200 {object} v1.Certificate
// @Param namespace path string true "Namespace"
// @Param name path string true "Name of the certificate"
// @Param certificate body v1.Certificate true "Updated Certificate"
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/certificates/{name} [put].
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

	var req v1.Certificate

	err = ginutil.BindJSON(ctx, &req)
	if err != nil {
		ginutil.HandleValidationError(ctx, "body", "", err, false)

		return
	}

	updated, err := c.certificateUsecase.UpdateCertificate(
		ctx.Request.Context(), namespace, name, &req,
	)
	if err != nil {
		c.logger.Error("failed to update certificate", "name", name, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while updating the certificate.")

		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// Delete deletes a certificate by its name.
//
// @Summary  Delete Certificate
// @Tags certificate
// @Description Delete a certificate by its name.
// @Param namespace path string true "Namespace"
// @Param name path string true "Name of the certificate"
// @Success 204
// @Failure 400 {object} map[string]any
// @Failure 404 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/certificates/{name} [delete].
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

	err = c.certificateUsecase.DeleteCertificate(ctx.Request.Context(), namespace, name)
	if err != nil {
		c.logger.Error("failed to delete certificate", "name", name, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while deleting the certificate.")

		return
	}

	ctx.Status(http.StatusNoContent)
}
