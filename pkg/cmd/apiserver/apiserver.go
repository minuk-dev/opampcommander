// Package apiserver provides the command for the apiserver.
package apiserver

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/app"
)

// CommandOption contains the options for the apiserver command.
type CommandOption struct {
	// flags
	dbHost    string
	addr      string
	logLevel  string
	logFormat string

	// internal
	app *app.Server
}

// NewCommand creates a new apiserver command.
func NewCommand(opt CommandOption) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "apiserver",
		Short: "apiserver",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := opt.Prepare(cmd, args)
			if err != nil {
				return err
			}

			err = opt.Run(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opt.addr, "addr", ":8080", "server address")
	cmd.Flags().StringVar(&opt.dbHost, "db-host", "localhost:2379", "etcd host")
	cmd.Flags().StringVar(&opt.logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().StringVar(&opt.logFormat, "log-format", "json", "log format (json, text)")

	return cmd
}

// Prepare prepares the command.
func (opt *CommandOption) Prepare(_ *cobra.Command, _ []string) error {
	logLevel := toSlogLevel(opt.logLevel)
	opt.app = app.NewServer(app.ServerSettings{
		Addr:      opt.addr,
		EtcdHosts: []string{opt.dbHost},
		LogLevel:  logLevel,
		LogFormat: app.LogFormat(opt.logFormat),
	})

	return nil
}

// Run runs the command.
func (opt *CommandOption) Run(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	err := opt.app.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run the server: %w", err)
	}

	return nil
}

func toSlogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
