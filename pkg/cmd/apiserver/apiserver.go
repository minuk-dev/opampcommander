// Package apiserver provides the command for the apiserver.
package apiserver

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/minuk-dev/opampcommander/pkg/apiserver"
	appconfig "github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/observability"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
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
		Sharding       struct {
			Enabled bool `mapstructure:"enabled"`
		} `mapstructure:"sharding"`
	} `mapstructure:"database"`
	ServiceName string `mapstructure:"serviceName"`
	Event       struct {
		Type  string `mapstructure:"type"`
		Kafka struct {
			Brokers []string `mapstructure:"brokers"`
			Topic   string   `mapstructure:"topic"`
		}
		Direct struct {
			SubProtocol      string `mapstructure:"subProtocol"`
			ListenAddress    string `mapstructure:"listenAddress"`
			AdvertiseAddress string `mapstructure:"advertiseAddress"`
			AuthToken        string `mapstructure:"authToken"`
		} `mapstructure:"direct"`
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
		Basic struct {
			Pepper string `mapstructure:"pepper"`
		} `mapstructure:"basic"`
		JWT struct {
			Issuer        string        `mapstructure:"issuer"`
			Expire        time.Duration `mapstructure:"expire"`
			RefreshExpire time.Duration `mapstructure:"refreshExpire"`
			Secret        string        `mapstructure:"secret"`
			Audience      []string      `mapstructure:"audience"`
		}
		Type   string `mapstructure:"type"`
		OAuth2 struct {
			Provider             string   `mapstructure:"provider"`
			ClientID             string   `mapstructure:"clientId"`
			ClientSecret         string   `mapstructure:"clientSecret"`
			RedirectURI          string   `mapstructure:"redirectUri"`
			AllowedRedirectHosts []string `mapstructure:"allowedRedirectHosts"`
			State                struct {
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
	Bootstrap struct {
		Dir                    string `mapstructure:"dir"`
		RemoteConfigSchemaDir  string `mapstructure:"remoteConfigSchemaDir"`
		RemoteConfigSchemaLoad string `mapstructure:"remoteConfigSchemaLoad"`
		DefaultNamespace       string `mapstructure:"defaultNamespace"`
		DefaultRole            string `mapstructure:"defaultRole"`
	} `mapstructure:"bootstrap"`

	MetricsBackend struct {
		Type          string        `mapstructure:"type"`
		Address       string        `mapstructure:"address"`
		DefaultWindow time.Duration `mapstructure:"defaultWindow"`
	} `mapstructure:"metricsBackend"`

	Liveness struct {
		FlushInterval   time.Duration `mapstructure:"flushInterval"`
		FlushStaleAfter time.Duration `mapstructure:"flushStaleAfter"`
		FlushBatchSize  int           `mapstructure:"flushBatchSize"`
		PersistThrottle time.Duration `mapstructure:"persistThrottle"`
		Redis           struct {
			Enabled        bool          `mapstructure:"enabled"`
			Endpoints      []string      `mapstructure:"endpoints"`
			MasterName     string        `mapstructure:"masterName"`
			Username       string        `mapstructure:"username"`
			Password       string        `mapstructure:"password"`
			DB             int           `mapstructure:"db"`
			TLS            bool          `mapstructure:"tls"`
			DialTimeout    time.Duration `mapstructure:"dialTimeout"`
			CommandTimeout time.Duration `mapstructure:"commandTimeout"`
			TTL            time.Duration `mapstructure:"ttl"`
		} `mapstructure:"redis"`
	} `mapstructure:"liveness"`

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
	cmd.Flags().String("database.type", "inmemory", "database type (inmemory, mongodb)")
	cmd.Flags().StringSlice("database.endpoints", []string{"mongodb://localhost:27017"}, "database endpoints")
	cmd.Flags().Duration("database.connectTimeout", 10*time.Second, "database connection timeout")
	cmd.Flags().String("database.databaseName", "opampcommander", "database name")
	cmd.Flags().Bool("database.ddlAuto", false, "automatically create database schema")
	cmd.Flags().Bool("database.sharding.enabled", false,
		"enable sharding-aware schema management (enableSharding + shardCollection); requires database.ddlAuto "+
			"and endpoints pointing at mongos routers")
	cmd.Flags().String("serviceName", "opampcommander", "service name for observability")
	cmd.Flags().String("event.type", "inmemory", "event protocol type (inmemory, kafka)")
	cmd.Flags().Bool("event.enabled", false, "enable event communication")
	cmd.Flags().StringSlice("event.kafka.brokers", []string{"localhost:9092"}, "Kafka broker addresses")
	cmd.Flags().String("event.kafka.topic", "opampcommander.events", "Kafka topic name")
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
	cmd.Flags().String("auth.basic.pepper", "",
		"server-side secret mixed into basic-auth password hashes; "+
			"set a long random value to enable DB-backed basic-auth users (empty disables them)")
	cmd.Flags().String("auth.jwt.issuer", "opampcommander", "JWT issuer")
	//nolint:mnd
	cmd.Flags().Duration("auth.jwt.expire", 30*time.Minute, "JWT access token expiration duration")
	//nolint:mnd
	cmd.Flags().Duration("auth.jwt.refreshExpire", 7*24*time.Hour,
		"JWT refresh token expiration duration (0 disables refresh tokens)")
	cmd.Flags().String("auth.jwt.secret", "", "JWT signing secret")
	cmd.Flags().StringSlice("auth.jwt.audience", []string{"opampcommander"}, "JWT audience")
	cmd.Flags().String("auth.type", "oauth2", "authentication type")
	cmd.Flags().String("auth.oauth2.provider", "", "OAuth2 provider URL")
	cmd.Flags().String("auth.oauth2.clientId", "", "OAuth2 client ID")
	cmd.Flags().String("auth.oauth2.clientSecret", "", "OAuth2 client secret")
	cmd.Flags().String("auth.oauth2.redirectUri", "", "OAuth2 redirect URL")
	cmd.Flags().StringSlice(
		"auth.oauth2.allowedRedirectHosts",
		nil,
		"additional hosts the OAuth2 authcode endpoint accepts as redirect "+
			"targets (loopback hosts are always allowed)",
	)
	cmd.Flags().String("auth.oauth2.state.mode", "jwt", "OAuth2 state mode (jwt)")
	cmd.Flags().String("auth.oauth2.state.jwt.secret", "", "OAuth2 state JWT secret")

	cmd.Flags().String("bootstrap.dir", "",
		"directory of initial manifest YAML files to seed on startup "+
			"(empty disables; the container image sets BOOTSTRAP_DIR=/etc/opampcommander/initial)")
	cmd.Flags().String("bootstrap.remoteConfigSchemaDir", "",
		"directory of the pre-built RemoteConfigSchema library "+
			"(empty defaults to <bootstrap.dir>/remoteconfigschema)")
	cmd.Flags().String("bootstrap.remoteConfigSchemaLoad", "latest",
		"which schemas to seed on startup: latest (newest per distribution), all, or none")
	cmd.Flags().String("bootstrap.defaultNamespace", "default",
		"namespace agents without a service.namespace are placed in, and where the default role is granted")
	cmd.Flags().String("bootstrap.defaultRole", "default",
		"name of the built-in role auto-granted to every user")
	cmd.Flags().String("metricsBackend.type", "none",
		"metrics backend for endpoint-throughput queries (none, prometheus)")
	cmd.Flags().String("metricsBackend.address", "",
		"base URL of the Prometheus-compatible HTTP API (required when metricsBackend.type=prometheus)")
	cmd.Flags().Duration("metricsBackend.defaultWindow", 5*time.Minute,
		"default rate window for endpoint-throughput queries")
	cmd.Flags().Duration("liveness.flushInterval", 30*time.Second,
		"how often agent liveness absorbed by the fast tier is written through to the database")
	cmd.Flags().Duration("liveness.flushStaleAfter", 30*time.Second,
		"how far behind a stored agent document must fall before the write-behind flush claims it; "+
			"flushInterval + flushStaleAfter must stay inside the 60s staleness budget")
	cmd.Flags().Int("liveness.flushBatchSize", 2000,
		"maximum agents written by one liveness flush cycle")
	cmd.Flags().Duration("liveness.persistThrottle", 0,
		"minimum interval between database writes for an agent whose only change is that it is still alive "+
			"(0 = 10s without a shared fast tier, the 60s staleness budget with one)")
	cmd.Flags().Bool("liveness.redis.enabled", false,
		"use Redis as a shared agent-liveness fast tier; optional accelerator, "+
			"the server falls back to node-local state and the database when it is unavailable")
	cmd.Flags().StringSlice("liveness.redis.endpoints", nil,
		"Redis addresses; one for a single server, several for a cluster")
	cmd.Flags().String("liveness.redis.masterName", "",
		"Redis Sentinel master name (selects Sentinel mode)")
	cmd.Flags().String("liveness.redis.username", "", "Redis username")
	cmd.Flags().String("liveness.redis.password", "", "Redis password")
	cmd.Flags().Int("liveness.redis.db", 0, "Redis logical database index (ignored in cluster mode)")
	cmd.Flags().Bool("liveness.redis.tls", false, "connect to Redis over TLS")
	cmd.Flags().Duration("liveness.redis.dialTimeout", 2*time.Second, "Redis connection timeout")
	cmd.Flags().Duration("liveness.redis.commandTimeout", 200*time.Millisecond,
		"per-command Redis timeout; kept short so a slow Redis degrades to the database "+
			"instead of holding up agent messages")
	cmd.Flags().Duration("liveness.redis.ttl", 120*time.Second,
		"how long a Redis liveness record survives unrefreshed; must exceed the 90s staleness window")

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

	// viper's StringSlice env-var support is famously fragile — without a
	// decode hook, `AUTH_OAUTH2_ALLOWEDREDIRECTHOSTS=a,b,c` lands as a
	// single-element slice. Adding StringToSliceHookFunc(",") makes every
	// `[]string` mapstructure field accept comma-separated env values,
	// while preserving native YAML list semantics. The Duration hook is
	// preserved for the existing time.Duration fields.
	err = opt.viper.Unmarshal(opt, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	))
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
	livenessSettings := opt.livenessSettings()

	// Fail at startup rather than in a degraded state nobody notices: a flush
	// interval too close to the staleness window, or a Redis fast tier that cannot
	// be reached as configured, are both silently harmful at runtime.
	err := livenessSettings.Validate()
	if err != nil {
		return fmt.Errorf("invalid liveness configuration: %w", err)
	}

	opt.app = apiserver.New(appconfig.ServerSettings{
		Address:  opt.Address,
		ServerID: agentmodel.ServerID(opt.ServerID),
		DatabaseSettings: appconfig.DatabaseSettings{
			Type:           appconfig.DatabaseType(opt.Database.Type),
			Endpoints:      opt.Database.Endpoints,
			ConnectTimeout: opt.Database.ConnectTimeout,
			DatabaseName:   opt.Database.DatabaseName,
			DDLAuto:        opt.Database.DDLAuto,
			Sharding: appconfig.ShardingSettings{
				Enabled: opt.Database.Sharding.Enabled,
			},
		},
		Security: security.Config{
			AdminSettings: security.AdminSettings{
				Username: opt.Auth.Admin.Username,
				Password: opt.Auth.Admin.Password,
				Email:    opt.Auth.Admin.Email,
			},
			BasicAuthSettings: security.BasicAuthSettings{
				Pepper: opt.Auth.Basic.Pepper,
			},
			JWTSettings: security.JWTSettings{
				Issuer:            opt.Auth.JWT.Issuer,
				Expiration:        opt.Auth.JWT.Expire,
				RefreshExpiration: opt.Auth.JWT.RefreshExpire,
				SigningKey:        opt.Auth.JWT.Secret,
				Audience:          opt.Auth.JWT.Audience,
			},
			OAuthSettings: &security.OAuthSettings{
				ClientID:             opt.Auth.OAuth2.ClientID,
				Secret:               opt.Auth.OAuth2.ClientSecret,
				CallbackURL:          opt.Auth.OAuth2.RedirectURI,
				AllowedRedirectHosts: opt.Auth.OAuth2.AllowedRedirectHosts,
				JWTSettings: security.JWTSettings{
					Issuer:            opt.Auth.OAuth2.State.JWT.Issuer,
					Expiration:        opt.Auth.OAuth2.State.JWT.Expire,
					RefreshExpiration: 0,
					SigningKey:        opt.Auth.OAuth2.State.JWT.Secret,
					Audience:          opt.Auth.OAuth2.State.JWT.Audience,
				},
			},
		},
		EventSettings: appconfig.EventSettings{
			ProtocolType: appconfig.EventProtocolType(opt.Event.Type),
			KafkaSettings: appconfig.KafkaSettings{
				Brokers: opt.Event.Kafka.Brokers,
				Topic:   opt.Event.Kafka.Topic,
			},
			DirectSettings: appconfig.DirectSettings{
				SubProtocol:      appconfig.DirectSubProtocol(opt.Event.Direct.SubProtocol),
				ListenAddress:    opt.Event.Direct.ListenAddress,
				AdvertiseAddress: opt.Event.Direct.AdvertiseAddress,
				AuthToken:        opt.Event.Direct.AuthToken,
			},
		},
		ManagementSettings: appconfig.ManagementSettings{
			Address: opt.Management.Address,
			Observability: observability.Config{
				ServiceName: opt.ServiceName,
				Metric: observability.MetricSettings{
					Enabled: opt.Management.Metric.Enabled,
					Type:    observability.MetricType(opt.Management.Metric.Type),
					MetricSettingsForPrometheus: observability.MetricSettingsForPrometheus{
						Path: opt.Management.Metric.Prometheus.Path,
					},
					MetricSettingsForOpenTelemetry: observability.MetricsSettingsForOpenTelemetry{
						Endpoint: opt.Management.Metric.OpenTelemetry.Endpoint,
					},
				},
				Log: observability.LogSettings{
					Enabled: opt.Management.Log.Enabled,
					Level:   toSlogLevel(opt.Management.Log.Level),
					Format:  observability.LogFormat(opt.Management.Log.Format),
				},
				Trace: observability.TraceSettings{
					Enabled:              opt.Management.Trace.Enabled,
					Protocol:             observability.TraceProtocol(opt.Management.Trace.Protocol),
					Compression:          opt.Management.Trace.Compression,
					CompressionAlgorithm: observability.TraceCompressionAlgorithm(opt.Management.Trace.CompressionAlgorithm),
					Insecure:             opt.Management.Trace.Insecure,
					Headers:              opt.Management.Trace.Headers,
					Sampler:              observability.TraceSampler(opt.Management.Trace.Sampler),
					SamplerRatio:         opt.Management.Trace.SamplerRatio,
					Endpoint:             opt.Management.Trace.Endpoint,
				},
			},
		},
		CacheSettings:    appconfig.DefaultCacheSettings(),
		LivenessSettings: livenessSettings,
		BootstrapSettings: appconfig.BootstrapSettings{
			Dir:                    opt.Bootstrap.Dir,
			RemoteConfigSchemaDir:  opt.Bootstrap.RemoteConfigSchemaDir,
			RemoteConfigSchemaLoad: opt.Bootstrap.RemoteConfigSchemaLoad,
			DefaultNamespace:       defaultString(opt.Bootstrap.DefaultNamespace, agentmodel.DefaultNamespaceName),
			DefaultRole:            defaultString(opt.Bootstrap.DefaultRole, usermodel.RoleDefault),
		},
		MetricsBackend: appconfig.MetricsBackendSettings{
			Type:          appconfig.MetricsBackendType(opt.MetricsBackend.Type),
			Address:       opt.MetricsBackend.Address,
			DefaultWindow: opt.MetricsBackend.DefaultWindow,
		},
		RBACModelPath: "",
	})

	return nil
}

// defaultString returns value when non-empty, otherwise fallback.
func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

// Run runs the command.
func (opt *CommandOption) Run(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	err := opt.app.Run(ctx)
	if err != nil {
		visualizedStr, visualizedErr := apiserver.VisualizeError(err)
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

// livenessSettings maps the parsed liveness options onto the server settings,
// falling back to the defaults for anything left unset.
func (opt *CommandOption) livenessSettings() appconfig.LivenessSettings {
	settings := appconfig.DefaultLivenessSettings()

	if opt.Liveness.FlushInterval > 0 {
		settings.FlushInterval = opt.Liveness.FlushInterval
	}

	if opt.Liveness.FlushStaleAfter > 0 {
		settings.FlushStaleAfter = opt.Liveness.FlushStaleAfter
	}

	if opt.Liveness.FlushBatchSize > 0 {
		settings.FlushBatchSize = opt.Liveness.FlushBatchSize
	}

	settings.PersistThrottle = opt.Liveness.PersistThrottle

	redis := opt.Liveness.Redis
	settings.Redis.Enabled = redis.Enabled
	settings.Redis.Endpoints = redis.Endpoints
	settings.Redis.MasterName = redis.MasterName
	settings.Redis.Username = redis.Username
	settings.Redis.Password = redis.Password
	settings.Redis.DB = redis.DB
	settings.Redis.TLS = redis.TLS

	if redis.DialTimeout > 0 {
		settings.Redis.DialTimeout = redis.DialTimeout
	}

	if redis.CommandTimeout > 0 {
		settings.Redis.CommandTimeout = redis.CommandTimeout
	}

	if redis.TTL > 0 {
		settings.Redis.TTL = redis.TTL
	}

	return settings
}
