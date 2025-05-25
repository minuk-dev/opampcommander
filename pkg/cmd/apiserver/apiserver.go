// Package apiserver provides the command for the apiserver.
package apiserver

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/minuk-dev/opampcommander/pkg/app"
)

// CommandOption contains the options for the apiserver command.
type CommandOption struct {
	configFilename string

	// flags
	dbType    string
	dbHosts   []string
	addr      string
	logLevel  string
	logFormat string
	// flags auth
	authEnabled              bool
	authType                 string
	authOauth2Provider       string
	authOauth2ClientID       string
	authOauth2ClientSecret   string
	authOauth2RedirectURL    string
	authOauth2StateMode      string
	authOauth2StateJWTSecret string

	// viper
	viper *viper.Viper

	// internal
	app *app.Server
}

// NewCommand creates a new apiserver command.
func NewCommand(opt CommandOption) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "apiserver",
		Short: "apiserver",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := opt.Init(cmd, args)
			if err != nil {
				return fmt.Errorf("failed to initialize command: %w", err)
			}
			return nil
		},
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

	cmd.PersistentFlags().StringVar(&opt.configFilename, "config", "", "config file (default is $HOME/.config/opampcommander/apiserver/config.yaml)")
	cmd.Flags().StringVar(&opt.addr, "address", ":8080", "server address")
	cmd.Flags().StringVar(&opt.dbType, "database.type", "etcd", "etcd")
	cmd.Flags().StringSliceVar(&opt.dbHosts, "database.endpoints", []string{"localhost:2379"}, "etcd host")
	cmd.Flags().StringVar(&opt.logLevel, "log.level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().StringVar(&opt.logFormat, "log.format", "json", "log format (json, text)")
	cmd.Flags().BoolVar(&opt.authEnabled, "auth.enabled", false, "enable authentication")
	cmd.Flags().StringVar(&opt.authType, "auth.type", "oauth2", "authentication type")
	cmd.Flags().StringVar(&opt.authOauth2Provider, "auth.oauth2.provider", "", "OAuth2 provider URL")
	cmd.Flags().StringVar(&opt.authOauth2ClientID, "auth.oauth2.clientId", "", "OAuth2 client ID")
	cmd.Flags().StringVar(&opt.authOauth2ClientSecret, "auth.oauth2.clientSecret", "", "OAuth2 client secret")
	cmd.Flags().StringVar(&opt.authOauth2RedirectURL, "auth.oauth2.redirectUri", "", "OAuth2 redirect URL")
	cmd.Flags().StringVar(&opt.authOauth2StateMode, "auth.oauth2.state.mode", "jwt", "OAuth2 state mode (jwt)")
	cmd.Flags().StringVar(&opt.authOauth2StateJWTSecret, "auth.oauth2.state.jwt.secret", "", "OAuth2 state JWT secret")
	opt.viper.BindPFlags(cmd.Flags())

	return cmd
}

func (opt *CommandOption) Init(_ *cobra.Command, _ []string) error {
	if opt.configFilename != "" {
		viper.SetConfigFile(opt.configFilename)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}

		opt.viper.AddConfigPath(filepath.Join(home, ".config", "opampcommander", "apiserver"))
		opt.viper.SetConfigName("config")
		opt.viper.SetConfigType("yaml")
	}

	opt.viper.AutomaticEnv() // read in environment variables that match

	if err := opt.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return nil
}

// Prepare prepares the command.
func (opt *CommandOption) Prepare(_ *cobra.Command, _ []string) error {
	logLevel := toSlogLevel(opt.logLevel)
	opt.app = app.NewServer(app.ServerSettings{
		Addr:      opt.addr,
		EtcdHosts: opt.dbHosts,
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
