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
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver"
	appconfig "github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// CommandOption contains the options for the apiserver command.
type CommandOption struct {
	configFilename string

	// flags
	Address  string `mapstructure:"address"`
	ServerID string `mapstructure:"serverId"`
	Database struct {
		Type           string        `mapstructure:"type"`
		Endpoints      []string      `mapstructure:"endpoints"`
		ConnectTimeout time.Duration `mapstructure:"connectTimeout"`
		DatabaseName   string        `mapstructure:"databaseName"`
		DDLAuto        bool          `mapstructure:"ddlAuto"`
	} `mapstructure:"database"`
	ServiceName string `mapstructure:"serviceName"`
	Event       struct {
		Type    string `mapstructure:"type"`
		Enabled bool   `mapstructure:"enabled"`
		NATS    struct {
			Endpoint      string `mapstructure:"endpoint"`
			SubjectPrefix string `mapstructure:"subjectPrefix"`
		}
	} `mapstructure:"event"`
	Management struct {
		Address string `mapstructure:"address"`
		Metric  struct {
			Enabled    bool   `mapstructure:"enabled"`
			Type       string `mapstructure:"type"`
			Prometheus struct {
				Path string `mapstructure:"path"`
			} `mapstructure:"prometheus"`
			OpenTelemetry struct {
				Endpoint string `mapstructure:"endpoint"`
			} `mapstructure:"openTelemetry"`
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
	} `mapstructure:"management"`
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
//nolint:funlen,mnd
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
	cmd.Flags().String("serverId", "", "server ID (default is hostname, can be overridden by SERVER_ID env var)")
	cmd.Flags().String("database.type", "mongodb", "database type (mongodb)")
	cmd.Flags().StringSlice("database.endpoints", []string{"mongodb://localhost:27017"}, "database endpoints")
	cmd.Flags().Duration("database.connectTimeout", 10*time.Second, "database connection timeout")
	cmd.Flags().String("database.databaseName", "opampcommander", "database name")
	cmd.Flags().Bool("database.ddlAuto", false, "automatically create database schema")
	cmd.Flags().String("serviceName", "opampcommander", "service name for observability")
	cmd.Flags().String("event.type", "nats", "event protocol type (nats)")
	cmd.Flags().Bool("event.enabled", false, "enable event communication")
	cmd.Flags().String("event.nats.endpoint", "nats://localhost:4222", "NATS server endpoint")
	cmd.Flags().String("event.nats.subjectPrefix", "test.opampcommander.", "NATS subject prefix")
	cmd.Flags().String("management.address", "localhost:9090", "management server address")
	cmd.Flags().Bool("management.metric.enabled", false, "enable metrics")
	cmd.Flags().String("management.metric.type", "prometheus", "metric type (prometheus, opentelemetry)")
	cmd.Flags().String("management.metric.prometheus.path", "/metrics", "Prometheus metrics path")
	cmd.Flags().String("management.metric.openTelemetry.endpoint", "localhost:4317", "OpenTelemetry metrics endpoint")
	cmd.Flags().Bool("management.log.enabled", true, "enable logging")
	cmd.Flags().String("management.log.level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().String("management.log.format", "text", "log format (json, text)")
	cmd.Flags().Bool("management.trace.enabled", false, "enable tracing")
	cmd.Flags().String(
		"management.trace.endpoint",
		"grpc://localhost:4317",
		"tracing endpoint (for OpenTelemetry, Jaeger, etc.)",
	)
	cmd.Flags().String("management.trace.protocol", "grpc", "tracing protocol (grpc, http/protobuf, http/json)")
	cmd.Flags().Bool("management.trace.compression", false, "enable compression for tracing")
	cmd.Flags().String("management.trace.compressionAlgorithm", "gzip", "compression algorithm for tracing (gzip)")
	cmd.Flags().Bool("management.trace.insecure", false, "use insecure connection for tracing")
	cmd.Flags().StringToString("management.trace.headers", nil, "headers to be sent with tracing requests")
	cmd.Flags().String("management.trace.sampler", "always", "tracing sampler (always, never, probability)")
	cmd.Flags().Float64(
		"management.trace.samplerRatio",
		1.0,
		"sampling ratio for traceidratio and parentbased_traceidratio samplers",
	)
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
	// SERVER_ID env var can override serverId
	opt.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // replace '.' with '_' for environment variables
	opt.viper.AutomaticEnv()                                   // read in environment variables that match

	err = opt.viper.Unmarshal(opt)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// If serverID is not set, use hostname as default
	if opt.ServerID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("failed to get hostname: %w", err)
		}

		opt.ServerID = hostname
	}

	return nil
}

// Prepare prepares the command.
//
//nolint:funlen // Configuration parsing requires many steps
func (opt *CommandOption) Prepare(_ *cobra.Command, _ []string) error {
	opt.app = apiserver.New(appconfig.ServerSettings{
		Address:  opt.Address,
		ServerID: appconfig.ServerID(opt.ServerID),
		DatabaseSettings: appconfig.DatabaseSettings{
			Type:           appconfig.DatabaseType(opt.Database.Type),
			Endpoints:      opt.Database.Endpoints,
			ConnectTimeout: opt.Database.ConnectTimeout,
			DatabaseName:   opt.Database.DatabaseName,
			DDLAuto:        opt.Database.DDLAuto,
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
		EventSettings: appconfig.EventSettings{
			Enabled:      opt.Event.Enabled,
			ProtocolType: appconfig.EventProtocolType(opt.Event.Type),
			NATS: appconfig.NATSSettings{
				Endpoint:      opt.Event.NATS.Endpoint,
				SubjectPrefix: opt.Event.NATS.SubjectPrefix,
			},
		},
		ManagementSettings: appconfig.ManagementSettings{
			Address: opt.Management.Address,
			ObservabilitySettings: appconfig.ObservabilitySettings{
				ServiceName: opt.ServiceName,
				Metric: appconfig.MetricSettings{
					Enabled: opt.Management.Metric.Enabled,
					Type:    appconfig.MetricType(opt.Management.Metric.Type),
					MetricSettingsForPrometheus: appconfig.MetricSettingsForPrometheus{
						Path: opt.Management.Metric.Prometheus.Path,
					},
					MetricSettingsForOpenTelemetry: appconfig.MetricsSettingsForOpenTelemetry{
						Endpoint: opt.Management.Metric.OpenTelemetry.Endpoint,
					},
				},
				Log: appconfig.LogSettings{
					Enabled: opt.Management.Log.Enabled,
					Level:   toSlogLevel(opt.Management.Log.Level),
					Format:  appconfig.LogFormat(opt.Management.Log.Format),
				},
				Trace: appconfig.TraceSettings{
					Enabled:              opt.Management.Trace.Enabled,
					Protocol:             appconfig.TraceProtocol(opt.Management.Trace.Protocol),
					Compression:          opt.Management.Trace.Compression,
					CompressionAlgorithm: appconfig.TraceCompressionAlgorithm(opt.Management.Trace.CompressionAlgorithm),
					Insecure:             opt.Management.Trace.Insecure,
					Headers:              opt.Management.Trace.Headers,
					Sampler:              appconfig.TraceSampler(opt.Management.Trace.Sampler),
					SamplerRatio:         opt.Management.Trace.SamplerRatio,
					Endpoint:             opt.Management.Trace.Endpoint,
				},
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
		visualizedStr, visualizedErr := fx.VisualizeError(err)
		if visualizedErr != nil {
			return fmt.Errorf("failed to visualize error of the server: %w", err)
		}

		cmd.PrintErr(visualizedStr)
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
