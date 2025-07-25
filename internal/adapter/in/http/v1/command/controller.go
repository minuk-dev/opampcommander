// Package command provides the command controller for the opampcommander.
package command

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	commandv1 "github.com/minuk-dev/opampcommander/api/v1/command"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
)

// Controller is a struct that implements the command controller.
type Controller struct {
	logger *slog.Logger
	// usecases
	commandUsecase applicationport.CommandLookUpUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	commandUsecase applicationport.CommandLookUpUsecase,
	logger *slog.Logger,
) *Controller {
	controller := &Controller{
		logger:         logger,
		commandUsecase: commandUsecase,
	}

	return controller
}

// RoutesInfo returns the routes information for the command controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/api/v1/commands/:id",
			Handler:     "http.v1.command.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      "GET",
			Path:        "/api/v1/commands",
			Handler:     "http.v1.command.List",
			HandlerFunc: c.List,
		},
	}
}

// Get retrieves a command by its ID.
//
// @Summary  Get Command
// @Tags command
// @Description  Retrieve a command by its ID.
// @Accept  json
// @Produce json
// @Param   id  path  string  true  "Command ID"
// @Success 200 {object} CommandAudit
// @Failure 400 {object} map[string]any "Invalid command ID"
// @Failure 500 {object} map[string]any "Failed to get command"
// @Router /api/v1/commands/{id} [get].
func (c *Controller) Get(ctx *gin.Context) {
	commandID := ctx.Param("id")

	commandIDUUID, err := uuid.Parse(commandID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid command ID"})

		return
	}

	command, err := c.commandUsecase.GetCommand(ctx, commandIDUUID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get command"})

		return
	}

	ctx.JSON(http.StatusOK, convertToAPIModel(command))
}

// List retrieves a list of commands.
//
// @Summary  List Commands
// @Tags command
// @Description  Retrieve a list of commands.
// @Accept  json
// @Produce json
// @Success 200 {array} CommandAudit
// @Failure 500 {object} map[string]any "Failed to list commands"
// @Router /api/v1/commands [get].
func (c *Controller) List(ctx *gin.Context) {
	limit, err := ginutil.GetQueryInt64(ctx, "limit", 0)
	if err != nil {
		c.logger.Error("failed to parse limit", "error", err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})

		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.commandUsecase.ListCommands(ctx, &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list commands"})

		return
	}

	ctx.JSON(
		http.StatusOK,
		commandv1.NewListResponse(
			lo.Map(response.Items, func(command *model.Command, _ int) commandv1.Audit {
				return convertToAPIModel(command)
			}),
			v1.ListMeta{
				RemainingItemCount: response.RemainingItemCount,
				Continue:           response.Continue,
			},
		),
	)
}

func convertToAPIModel(command *model.Command) commandv1.Audit {
	return commandv1.Audit{
		Kind:              string(command.Kind),
		ID:                command.ID.String(),
		TargetInstanceUID: command.TargetInstanceUID.String(),
		Data:              command.Data,
	}
}
