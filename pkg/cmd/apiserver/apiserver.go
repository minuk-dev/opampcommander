// Package apiserver provides the command for the apiserver.
package apiserver

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/minuk-dev/opampcommander/pkg/apiserver"
	appconfig "github.com/minuk-dev/opampcommander/pkg/apiserver/config"
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
	ServiceName string `mapstructure:"serviceName"`
	Metric      struct {
		Enabled  bool   `mapstructure:"enabled"`
		Type     string `mapstructure:"type"`
		Endpoint string `mapstructure:"endpoint"`
	}
	Log struct {
		Enabled bool   `mapstructure:"enabled"`
		Level   string `mapstructure:"level"`
		Format  string `mapstructure:"format"`
	} `mapstructure:"log"`
	Trace struct {
		Enabled              bool              `mapstructure:"enabled"`
		Protocol             string            `mapstructure:"protocol"`
		Compression          bool              `mapstructure:"compression"`
		CompressionAlgorithm string            `mapstructure:"compressionAlgorithm"`
		Insecure             bool              `mapstructure:"insecure"`
		Headers              map[string]string `mapstructure:"headers"`
		Endpoint             string            `mapstructure:"endpoint"`
		Sampler              string            `mapstructure:"sampler"`
		SamplerRatio         float64           `mapstructure:"samplerRatio"`
	} `mapstructure:"trace"`
	Auth struct {
		Enabled bool `mapstructure:"enabled"`
		Admin   struct {
			Username string `mapstructure:"username"`
			Password string `mapstructure:"password"`
			Email    string `mapstructure:"email"`
		} `mapstructure:"admin"`
		JWT struct {
			Issuer   string        `mapstructure:"issuer"`
			Expire   time.Duration `mapstructure:"expire"`
			Secret   string        `mapstructure:"secret"`
			Audience []string      `mapstructure:"audience"`
		}
		Type   string `mapstructure:"type"`
		OAuth2 struct {
			Provider     string `mapstructure:"provider"`
			ClientID     string `mapstructure:"clientId"`
			ClientSecret string `mapstructure:"clientSecret"`
			RedirectURI  string `mapstructure:"redirectUri"`
			State        struct {
				Mode string `mapstructure:"mode"`
				JWT  struct {
					Issuer   string        `mapstructure:"issuer"`
					Expire   time.Duration `mapstructure:"expire"`
					Secret   string        `mapstructure:"secret"`
					Audience []string      `mapstructure:"audience"`
				} `mapstructure:"jwt"`
			} `mapstructure:"state"`
		} `mapstructure:"oauth2"`
	} `mapstructure:"auth"`

	// viper
	viper *viper.Viper

	// internal
	app *apiserver.Server
}

// NewCommand creates a new apiserver command.
//
//nolint:funlen
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
	cmd.Flags().String("address", "localhost:8080", "server address")
	cmd.Flags().String("database.type", "mongodb", "database type (etcd, mongodb)")
	cmd.Flags().StringSlice("database.endpoints", []string{"mongodb://localhost:27017"}, "database endpoints")
	cmd.Flags().String("serviceName", "opampcommander", "service name for observability")
	cmd.Flags().Bool("metric.enabled", false, "enable metrics")
	cmd.Flags().String("metric.type", "prometheus", "metric type (prometheus, opentelemetry)")
	cmd.Flags().String("metric.endpoint", "http://localhost:8081/metrics",
		"metric endpoint (for prometheus, opentelemetry)")
	cmd.Flags().Bool("log.enabled", true, "enable logging")
	cmd.Flags().String("log.level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().String("log.format", "text", "log format (json, text)")
	cmd.Flags().Bool("trace.enabled", false, "enable tracing")
	cmd.Flags().String("trace.endpoint", "grpc://localhost:4317", "tracing endpoint (for OpenTelemetry, Jaeger, etc.)")
	cmd.Flags().String("trace.protocol", "grpc", "tracing protocol (grpc, http/protobuf, http/json)")
	cmd.Flags().Bool("trace.compression", false, "enable compression for tracing")
	cmd.Flags().String("trace.compressionAlgorithm", "gzip", "compression algorithm for tracing (gzip)")
	cmd.Flags().Bool("trace.insecure", false, "use insecure connection for tracing")
	cmd.Flags().StringToString("trace.headers", nil, "headers to be sent with tracing requests")
	cmd.Flags().String("trace.sampler", "always", "tracing sampler (always, never, probability)")
	cmd.Flags().Float64("trace.samplerRatio", 1.0, "sampling ratio for traceidratio and parentbased_traceidratio samplers")
	cmd.Flags().Bool("auth.enabled", false, "enable authentication")
	cmd.Flags().String("auth.admin.username", "admin", "admin username")
	cmd.Flags().String("auth.admin.password", "admin", "admin password")
	cmd.Flags().String("auth.admin.email", "admin@admin", "admin email")
	cmd.Flags().String("auth.jwt.issuer", "opampcommander", "JWT issuer")
	//nolint:mnd
	cmd.Flags().Duration("auth.jwt.expire", 30*time.Minute, "JWT expiration duration")
	cmd.Flags().String("auth.jwt.secret", "", "JWT signing secret")
	cmd.Flags().StringSlice("auth.jwt.audience", []string{"opampcommander"}, "JWT audience")
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
		opt.viper.SetConfigFile(opt.configFilename)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}

		opt.viper.AddConfigPath(filepath.Join(home, ".config", "opampcommander", "apiserver"))
		opt.viper.SetConfigName("config")
		opt.viper.SetConfigType("yaml")
	}

	_ = opt.viper.ReadInConfig()

	// Use environment variables
	// e.g. LOG_LEVEL=debug will set log.level to debug
	opt.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // replace '.' with '_' for environment variables
	opt.viper.AutomaticEnv()                                   // read in environment variables that match

	err = opt.viper.Unmarshal(opt)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// Prepare prepares the command.
func (opt *CommandOption) Prepare(_ *cobra.Command, _ []string) error {
	opt.app = apiserver.New(appconfig.ServerSettings{
		Address: opt.Address,
		DatabaseSettings: appconfig.DatabaseSettings{
			Type:      appconfig.DatabaseType(opt.Database.Type),
			Endpoints: opt.Database.Endpoints,
		},
		AuthSettings: appconfig.AuthSettings{
			AdminSettings: appconfig.AdminSettings{
				Username: opt.Auth.Admin.Username,
				Password: opt.Auth.Admin.Password,
				Email:    opt.Auth.Admin.Email,
			},
			JWTSettings: appconfig.JWTSettings{
				Issuer:     opt.Auth.JWT.Issuer,
				Expiration: opt.Auth.JWT.Expire,
				SigningKey: opt.Auth.JWT.Secret,
				Audience:   opt.Auth.JWT.Audience,
			},
			OAuthSettings: &appconfig.OAuthSettings{
				ClientID:    opt.Auth.OAuth2.ClientID,
				Secret:      opt.Auth.OAuth2.ClientSecret,
				CallbackURL: opt.Auth.OAuth2.RedirectURI,
				JWTSettings: appconfig.JWTSettings{
					Issuer:     opt.Auth.OAuth2.State.JWT.Issuer,
					Expiration: opt.Auth.OAuth2.State.JWT.Expire,
					SigningKey: opt.Auth.OAuth2.State.JWT.Secret,
					Audience:   opt.Auth.OAuth2.State.JWT.Audience,
				},
			},
		},
		ObservabilitySettings: appconfig.ObservabilitySettings{
			ServiceName: opt.ServiceName,
			Metric: appconfig.MetricSettings{
				Enabled:  opt.Metric.Enabled,
				Type:     appconfig.MetricType(opt.Metric.Type),
				Endpoint: opt.Metric.Endpoint,
			},
			Log: appconfig.LogSettings{
				Enabled: opt.Log.Enabled,
				Level:   toSlogLevel(opt.Log.Level),
				Format:  appconfig.LogFormat(opt.Log.Format),
			},
			Trace: appconfig.TraceSettings{
				Enabled:              opt.Trace.Enabled,
				Protocol:             appconfig.TraceProtocol(opt.Trace.Protocol),
				Compression:          opt.Trace.Compression,
				CompressionAlgorithm: appconfig.TraceCompressionAlgorithm(opt.Trace.CompressionAlgorithm),
				Insecure:             opt.Trace.Insecure,
				Headers:              opt.Trace.Headers,
				Sampler:              appconfig.TraceSampler(opt.Trace.Sampler),
				SamplerRatio:         opt.Trace.SamplerRatio,
				Endpoint:             opt.Trace.Endpoint,
			},
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
