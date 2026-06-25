// Package reconcile contains the controller for the generic reconcile endpoint, which
// re-enforces a domain object's invariants on demand by dispatching to a registered
// reconciler for the requested kind.
package reconcile

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

// Controller exposes the generic reconcile endpoint.
type Controller struct {
	logger *slog.Logger

	reconcileUsecase port.ReconcileManageUsecase
}

// NewController creates a new reconcile Controller.
func NewController(
	usecase port.ReconcileManageUsecase,
	logger *slog.Logger,
) *Controller {
	return &Controller{
		logger:           logger,
		reconcileUsecase: usecase,
	}
}

// RoutesInfo returns the routes for the reconcile controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/namespaces/:namespace/reconcile/:kind/:name",
			Handler:     "http.v1.reconcile.Reconcile",
			HandlerFunc: c.Reconcile,
		},
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/reconcile/kinds",
			Handler:     "http.v1.reconcile.ListKinds",
			HandlerFunc: c.ListKinds,
		},
	}
}

// Reconcile re-enforces the named resource's domain invariants by dispatching to the
// reconciler registered for the kind.
//
// @Summary  Reconcile a resource
// @Tags  reconcile
// @Description Re-run the side effects that normally fire on create/update for the resource.
// @Produce  json
// @Param  namespace path string true "Namespace"
// @Param  kind path string true "Resource kind (e.g. agentremoteconfig, agentgroup, agent)"
// @Param  name path string true "Resource name, or instance UID for kind=agent"
// @Success  204 "No Content"
// @Failure  400 {object} map[string]any
// @Failure  404 {object} map[string]any
// @Failure  500 {object} map[string]any
// @Router  /api/v1/namespaces/{namespace}/reconcile/{kind}/{name} [post].
func (c *Controller) Reconcile(ctx *gin.Context) {
	namespace, err := ginutil.ParseString(ctx, "namespace", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "namespace", ctx.Param("namespace"), err, true)

		return
	}

	kind, err := ginutil.ParseString(ctx, "kind", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "kind", ctx.Param("kind"), err, true)

		return
	}

	name, err := ginutil.ParseString(ctx, "name", true)
	if err != nil {
		ginutil.HandleValidationError(ctx, "name", ctx.Param("name"), err, true)

		return
	}

	err = c.reconcileUsecase.Reconcile(ctx.Request.Context(), kind, namespace, name)
	if err != nil {
		// HandleDomainError maps the application errors: an unknown kind / bad UID surfaces as
		// port.ErrInvalidArgument (400), a missing resource as port.ErrResourceNotExist (404).
		c.logger.Error("failed to reconcile resource",
			"kind", kind, "name", name, "error", err.Error())
		ginutil.HandleDomainError(ctx, err, "An error occurred while reconciling the resource.")

		return
	}

	ctx.Status(http.StatusNoContent)
}

// ListKinds returns the reconcilable kinds.
//
// @Summary  List reconcilable kinds
// @Tags  reconcile
// @Description List the resource kinds that support reconcile.
// @Produce  json
// @Success  200 {array} string
// @Router  /api/v1/reconcile/kinds [get].
func (c *Controller) ListKinds(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, c.reconcileUsecase.ReconcileKinds(ctx.Request.Context()))
}
