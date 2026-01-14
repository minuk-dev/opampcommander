// Package server provides the HTTP controller for managing servers.
package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	serverv1 "github.com/minuk-dev/opampcommander/api/v1/server"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that handles HTTP requests related to servers.
type Controller struct {
	logger *slog.Logger

	// usecases
	serverUsecase domainport.ServerUsecase
}

// NewController creates a new instance of the Controller struct.
func NewController(serverUsecase domainport.ServerUsecase) *Controller {
	return &Controller{
		logger:        slog.Default(),
		serverUsecase: serverUsecase,
	}
}

// RoutesInfo returns the routes information for the controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/api/v1/servers",
			Handler:     "http.v1.server.List",
			HandlerFunc: c.List,
		},
	}
}

// List handles the request to list all alive servers.
//
// @Summary List Servers
// @Tags server
// @Description  Retrieve a list of all alive servers.
// @Accept  json
// @Produce json
// @Success 200 {array} Server
// @Failure 500 {object} map[string]any
// @Router /api/v1/servers [get].
func (c *Controller) List(ctx *gin.Context) {
	servers, err := c.serverUsecase.ListServers(ctx.Request.Context())
	if err != nil {
		c.logger.Error("failed to list servers", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while listing servers.")

		return
	}

	serverResponse := serverv1.NewListResponse(
		lo.Map(servers, func(server *model.Server, _ int) serverv1.Server {
			return serverv1.Server{
				ID:              server.ID,
				LastHeartbeatAt: v1.NewTime(server.LastHeartbeatAt),
				Conditions:      mapConditionsToAPI(server.Conditions),
			}
		}),
		v1.ListMeta{
			RemainingItemCount: 0,
			Continue:           "",
		},
	)

	ctx.JSON(http.StatusOK, serverResponse)
}

// mapConditionsToAPI converts domain server conditions to API conditions.
func mapConditionsToAPI(conditions []model.ServerCondition) []serverv1.Condition {
	if len(conditions) == 0 {
		return nil
	}

	apiConditions := make([]serverv1.Condition, len(conditions))
	for i, condition := range conditions {
		apiConditions[i] = serverv1.Condition{
			Type:               serverv1.ConditionType(condition.Type),
			LastTransitionTime: v1.NewTime(condition.LastTransitionTime),
			Status:             serverv1.ConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	return apiConditions
}
