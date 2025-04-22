// Package command provides the command controller for the opampcommander.
package command

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commandv1 "github.com/minuk-dev/opampcommander/api/v1/command"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

// Controller is a struct that implements the command controller.
type Controller struct {
	logger *slog.Logger
	// usecases
	commandUsecase port.CommandUsecase
}

// NewController creates a new instance of Controller.
func NewController(
	commandUsecase port.CommandUsecase,
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
			Path:        "/v1/commands/:id",
			Handler:     "http.v1.command.Get",
			HandlerFunc: c.Get,
		},
		{
			Method:      "GET",
			Path:        "/v1/commands",
			Handler:     "http.v1.command.List",
			HandlerFunc: c.List,
		},
		{
			Method:      "POST",
			Path:        "/v1/commands/update-agent-config",
			Handler:     "http.v1.command.UpdateAgentConfig",
			HandlerFunc: c.UpdateAgentConfig,
		},
	}
}

// Get retrieves a command by its ID.
func (c *Controller) Get(ctx *gin.Context) {
	commandID := ctx.Param("id")
	if commandID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "command ID is required"})

		return
	}

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

	ctx.JSON(http.StatusOK, command)
}

// List retrieves a list of commands.
func (c *Controller) List(ctx *gin.Context) {
	commands, err := c.commandUsecase.ListCommands(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list commands"})

		return
	}

	ctx.JSON(http.StatusOK, commands)
}

// UpdateAgentConfig creates a new command to update the agent configuration.
func (c *Controller) UpdateAgentConfig(ctx *gin.Context) {
	var request commandv1.UpdateAgentConfigRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})

		return
	}

	command := model.NewUpdateAgentConfigCommand(request.TargetInstanceUID, request.RemoteConfig)
	if err := c.commandUsecase.SaveCommand(ctx, command); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save command"})

		return
	}

	ctx.JSON(http.StatusCreated, command)
}
