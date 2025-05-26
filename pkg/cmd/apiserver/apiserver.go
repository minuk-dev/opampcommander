// Package apiserver provides the command for the apiserver.
package apiserver

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/auth/github"
	"github.com/minuk-dev/opampcommander/pkg/app"
)

// CommandOption contains the options for the apiserver command.
type CommandOption struct {
	configFilename string

	// flags
	Address  string `mapstructure:"address"`
	Database struct {
		Type      string   `mapstructure:"type"`
		Endpoints []string `mapstructure:"endpoints"`
	} `mapstructure:"database"`
	Log struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"log"`
	Auth struct {
		Enabled bool   `mapstructure:"enabled"`
		Type    string `mapstructure:"type"`
		OAuth2  struct {
			Provider     string `mapstructure:"provider"`
			ClientID     string `mapstructure:"clientId"`
			ClientSecret string `mapstructure:"clientSecret"`
			RedirectURI  string `mapstructure:"redirectUri"`
			State        struct {
				Mode string `mapstructure:"mode"`
				JWT  struct {
					Secret string `mapstructure:"secret"`
				} `mapstructure:"jwt"`
			} `mapstructure:"state"`
		} `mapstructure:"oauth2"`
	} `mapstructure:"auth"`

	// viper
	viper *viper.Viper

	// internal
	app *app.Server
}

// NewCommand creates a new apiserver command.
func NewCommand(opt CommandOption) *cobra.Command {
	if opt.viper == nil {
		opt.viper = viper.New()
	}
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "apiserver",
		Short: "apiserver",
		PreRunE: func(cmd *cobra.Command, args []string) error {
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

	cmd.PersistentFlags().StringVar(&opt.configFilename, "config", "",
		"config file (default is $HOME/.config/opampcommander/apiserver/config.yaml)")
	cmd.Flags().String("address", ":8080", "server address")
	cmd.Flags().String("database.type", "etcd", "etcd")
	cmd.Flags().StringSlice("database.endpoints", []string{"localhost:2379"}, "etcd host")
	cmd.Flags().String("log.level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().String("log.format", "json", "log format (json, text)")
	cmd.Flags().Bool("auth.enabled", false, "enable authentication")
	cmd.Flags().String("auth.type", "oauth2", "authentication type")
	cmd.Flags().String("auth.oauth2.provider", "", "OAuth2 provider URL")
	cmd.Flags().String("auth.oauth2.clientId", "", "OAuth2 client ID")
	cmd.Flags().String("auth.oauth2.clientSecret", "", "OAuth2 client secret")
	cmd.Flags().String("auth.oauth2.redirectUri", "", "OAuth2 redirect URL")
	cmd.Flags().String("auth.oauth2.state.mode", "jwt", "OAuth2 state mode (jwt)")
	cmd.Flags().String("auth.oauth2.state.jwt.secret", "", "OAuth2 state JWT secret")

	return cmd
}

// Init initializes the command options.
func (opt *CommandOption) Init(cmd *cobra.Command, _ []string) error {
	err := opt.viper.BindPFlags(cmd.Flags())
	if err != nil {
		return fmt.Errorf("failed to bind flags: %w", err)
	}

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
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	err = opt.viper.Unmarshal(opt)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// Prepare prepares the command.
func (opt *CommandOption) Prepare(_ *cobra.Command, _ []string) error {
	logLevel := toSlogLevel(opt.Log.Level)
	opt.app = app.NewServer(app.ServerSettings{
		Addr:      opt.Address,
		EtcdHosts: opt.Database.Endpoints,
		LogLevel:  logLevel,
		LogFormat: app.LogFormat(opt.Log.Format),
		GithubOAuthSettings: &github.OAuthSettings{
			ClientID:    opt.Auth.OAuth2.ClientID,
			Secret:      opt.Auth.OAuth2.ClientSecret,
			CallbackURL: opt.Auth.OAuth2.RedirectURI,
		},
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
