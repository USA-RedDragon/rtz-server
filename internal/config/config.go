package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-errors/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP         HTTP         `json:"http"`
	Persistence  Persistence  `json:"persistence"`
	Registration Registration `json:"registration"`
	Auth         Auth         `json:"auth"`
	JWT          JWT          `json:"jwt"`
	Mapbox       Mapbox       `json:"mapbox"`
}

type JWT struct {
	Secret string `json:"secret"`
}

type Auth struct {
	Google Google `json:"google"`
	GitHub GitHub `json:"github"`
}

type Mapbox struct {
	SecretToken string `json:"secret_token" yaml:"secret_token"`
	PublicToken string `json:"public_token" yaml:"public_token"`
}

type Google struct {
	ClientID     string `json:"client_id" yaml:"client_id"`
	ClientSecret string `json:"client_secret" yaml:"client_secret"`
}

type GitHub struct {
	ClientID     string `json:"client_id" yaml:"client_id"`
	ClientSecret string `json:"client_secret" yaml:"client_secret"`
}

type Registration struct {
	Enabled bool `json:"enabled"`
}

type InitialAdmin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Persistence struct {
	Database string `json:"database"`
	Uploads  string `json:"uploads"`
}

type HTTPListener struct {
	IPV4Host string `json:"ipv4_host" yaml:"ipv4_host"`
	IPV6Host string `json:"ipv6_host" yaml:"ipv6_host"`
	Port     uint16 `json:"port"`
}

type Tracing struct {
	Enabled      bool   `json:"enabled"`
	OTLPEndpoint string `json:"otlp_endpoint" yaml:"otlp_endpoint"`
}

type PProf struct {
	Enabled bool `json:"enabled"`
}

type Metrics struct {
	HTTPListener
	Enabled bool `json:"enabled"`
}

type HTTP struct {
	HTTPListener
	Tracing
	BackendURL     string   `json:"backend_url" yaml:"backend_url"`
	PProf          PProf    `json:"pprof"`
	TrustedProxies []string `json:"trusted_proxies" yaml:"trusted_proxies"`
	Metrics        Metrics  `json:"metrics"`
	CORSHosts      []string `json:"cors_hosts" yaml:"cors_hosts"`
}

//nolint:golint,gochecknoglobals
var (
	ConfigFileKey             = "config"
	HTTPIPV4HostKey           = "http.ipv4_host"
	HTTPIPV6HostKey           = "http.ipv6_host"
	HTTPPortKey               = "http.port"
	HTTPTracingEnabledKey     = "http.tracing.enabled"
	HTTPTracingOTLPEndKey     = "http.tracing.otlp_endpoint"
	HTTPPProfEnabledKey       = "http.pprof.enabled"
	HTTPTrustedProxiesKey     = "http.trusted_proxies"
	HTTPMetricsEnabledKey     = "http.metrics.enabled"
	HTTPMetricsIPV4HostKey    = "http.metrics.ipv4_host"
	HTTPMetricsIPV6HostKey    = "http.metrics.ipv6_host"
	HTTPMetricsPortKey        = "http.metrics.port"
	HTTPCORSHostsKey          = "http.cors_hosts"
	HTTPBackendURLKey         = "http.backend_url"
	PersistenceDatabaseKey    = "persistence.database"
	PersistenceUploadsKey     = "persistence.uploads"
	RegistrationEnabledKey    = "registration.enabled"
	AuthGoogleClientIDKey     = "auth.google.client_id"
	AuthGoogleClientSecretKey = "auth.google.client_secret"
	AuthGitHubClientIDKey     = "auth.github.client_id"
	AuthGitHubClientSecretKey = "auth.github.client_secret"
	JWTSecretKey              = "jwt.secret"
	MapboxPublicTokenKey      = "mapbox.public_token"
	MapboxSecretTokenKey      = "mapbox.secret_token"
)

const (
	DefaultConfigPath          = "config.yaml"
	DefaultHTTPIPV4Host        = "0.0.0.0"
	DefaultHTTPIPV6Host        = "::"
	DefaultHTTPPort            = 8080
	DefaultHTTPMetricsIPV4Host = "127.0.0.1"
	DefaultHTTPMetricsIPV6Host = "::1"
	DefaultHTTPMetricsPort     = 8081
	DefaultPersistenceDatabase = "rtz.db"
	DefaultPersistenceUploads  = "uploads/"
	DefaultRegistrationEnabled = false
)

func RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(ConfigFileKey, "c", DefaultConfigPath, "Config file path")
	cmd.Flags().String(HTTPIPV4HostKey, DefaultHTTPIPV4Host, "HTTP server IPv4 host")
	cmd.Flags().String(HTTPIPV6HostKey, DefaultHTTPIPV6Host, "HTTP server IPv6 host")
	cmd.Flags().Uint16(HTTPPortKey, DefaultHTTPPort, "HTTP server port")
	cmd.Flags().Bool(HTTPTracingEnabledKey, false, "Enable Open Telemetry tracing")
	cmd.Flags().String(HTTPTracingOTLPEndKey, "", "Open Telemetry endpoint")
	cmd.Flags().Bool(HTTPPProfEnabledKey, false, "Enable pprof")
	cmd.Flags().StringSlice(HTTPTrustedProxiesKey, []string{}, "Comma-separated list of trusted proxies")
	cmd.Flags().Bool(HTTPMetricsEnabledKey, false, "Enable metrics server")
	cmd.Flags().String(HTTPMetricsIPV4HostKey, DefaultHTTPMetricsIPV4Host, "Metrics server IPv4 host")
	cmd.Flags().String(HTTPMetricsIPV6HostKey, DefaultHTTPMetricsIPV6Host, "Metrics server IPv6 host")
	cmd.Flags().Uint16(HTTPMetricsPortKey, DefaultHTTPMetricsPort, "Metrics server port")
	cmd.Flags().StringSlice(HTTPCORSHostsKey, []string{}, "Comma-separated list of CORS hosts")
	cmd.Flags().String(HTTPBackendURLKey, "", "Backend URL")
	cmd.Flags().String(PersistenceDatabaseKey, DefaultPersistenceDatabase, "Database file path")
	cmd.Flags().String(PersistenceUploadsKey, DefaultPersistenceUploads, "Uploads directory")
	cmd.Flags().Bool(RegistrationEnabledKey, DefaultRegistrationEnabled, "Enable registration")
	cmd.Flags().String(AuthGoogleClientIDKey, "", "Google OAuth client ID")
	cmd.Flags().String(AuthGoogleClientSecretKey, "", "Google OAuth client secret")
	cmd.Flags().String(AuthGitHubClientIDKey, "", "GitHub OAuth client ID")
	cmd.Flags().String(AuthGitHubClientSecretKey, "", "GitHub OAuth client secret")
	cmd.Flags().String(JWTSecretKey, "", "JWT signing secret")
	cmd.Flags().String(MapboxPublicTokenKey, "", "Mapbox public token")
	cmd.Flags().String(MapboxSecretTokenKey, "", "Mapbox secret token")
}

var (
	ErrorJWTSecretRequired         = errors.New("JWT secret is required")
	ErrorBackendURLRequired        = errors.New("Backend URL is required")
	ErrorOTLPEndpointRequired      = errors.New("OTLP endpoint is required when tracing is enabled")
	ErrorMapboxPublicTokenRequired = errors.New("Mapbox public token is required")
	ErrorMapboxSecretTokenRequired = errors.New("Mapbox secret token is required")
)

func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return ErrorJWTSecretRequired
	}
	if c.HTTP.BackendURL == "" {
		return ErrorBackendURLRequired
	}
	if c.HTTP.Tracing.Enabled && c.HTTP.Tracing.OTLPEndpoint == "" {
		return ErrorOTLPEndpointRequired
	}
	if c.Mapbox.PublicToken == "" {
		return ErrorMapboxPublicTokenRequired
	}
	if c.Mapbox.SecretToken == "" {
		return ErrorMapboxSecretTokenRequired
	}
	return nil
}

func LoadConfig(cmd *cobra.Command) (*Config, error) {
	var config Config

	// Load flags from envs
	ctx, cancel := context.WithCancelCause(cmd.Context())
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if ctx.Err() != nil {
			return
		}
		optName := strings.ReplaceAll(strings.ReplaceAll(strings.ToUpper(f.Name), "-", "_"), ".", "__")
		if val, ok := os.LookupEnv(optName); !f.Changed && ok {
			if err := f.Value.Set(val); err != nil {
				cancel(err)
			}
			f.Changed = true
		}
	})
	if ctx.Err() != nil {
		return &config, fmt.Errorf("failed to load env: %w", context.Cause(ctx))
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return &config, fmt.Errorf("failed to get config path: %w", err)
	}
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if errors.Is(err, os.ErrNotExist) {
			// This is okay, we default to reading a config file
		} else if err != nil {
			return &config, fmt.Errorf("failed to read config: %w", err)
		}

		if err := yaml.Unmarshal(data, &config); err != nil {
			return &config, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	err = overrideFlags(&config, cmd)
	if err != nil {
		return &config, fmt.Errorf("failed to override flags: %w", err)
	}

	// Defaults
	if config.HTTP.IPV4Host == "" {
		config.HTTP.IPV4Host = DefaultHTTPIPV4Host
	}
	if config.HTTP.IPV6Host == "" {
		config.HTTP.IPV6Host = DefaultHTTPIPV6Host
	}
	if config.HTTP.Port == 0 {
		config.HTTP.Port = DefaultHTTPPort
	}
	if config.HTTP.Metrics.IPV4Host == "" {
		config.HTTP.Metrics.IPV4Host = DefaultHTTPMetricsIPV4Host
	}
	if config.HTTP.Metrics.IPV6Host == "" {
		config.HTTP.Metrics.IPV6Host = DefaultHTTPMetricsIPV6Host
	}
	if config.HTTP.Metrics.Port == 0 {
		config.HTTP.Metrics.Port = DefaultHTTPMetricsPort
	}
	if config.Persistence.Database == "" {
		config.Persistence.Database = DefaultPersistenceDatabase
	}
	if config.Persistence.Uploads == "" {
		config.Persistence.Uploads = DefaultPersistenceUploads
	}

	return &config, nil
}

func overrideFlags(config *Config, cmd *cobra.Command) error {
	var err error
	if cmd.Flags().Changed(HTTPIPV4HostKey) {
		config.HTTP.IPV4Host, err = cmd.Flags().GetString(HTTPIPV4HostKey)
		if err != nil {
			return fmt.Errorf("failed to get HTTP IPv4 host: %w", err)
		}
	}

	if cmd.Flags().Changed(HTTPIPV6HostKey) {
		config.HTTP.IPV6Host, err = cmd.Flags().GetString(HTTPIPV6HostKey)
		if err != nil {
			return fmt.Errorf("failed to get HTTP IPv6 host: %w", err)
		}
	}

	if cmd.Flags().Changed(HTTPPortKey) {
		config.HTTP.Port, err = cmd.Flags().GetUint16(HTTPPortKey)
		if err != nil {
			return fmt.Errorf("failed to get HTTP port: %w", err)
		}
	}

	if cmd.Flags().Changed(HTTPPProfEnabledKey) {
		config.HTTP.PProf.Enabled, err = cmd.Flags().GetBool(HTTPPProfEnabledKey)
		if err != nil {
			return fmt.Errorf("failed to get pprof enabled: %w", err)
		}
	}

	if cmd.Flags().Changed(HTTPTrustedProxiesKey) {
		config.HTTP.TrustedProxies, err = cmd.Flags().GetStringSlice(HTTPTrustedProxiesKey)
		if err != nil {
			return fmt.Errorf("failed to get trusted proxies: %w", err)
		}
	}

	if cmd.Flags().Changed(HTTPMetricsEnabledKey) {
		config.HTTP.Metrics.Enabled, err = cmd.Flags().GetBool(HTTPMetricsEnabledKey)
		if err != nil {
			return fmt.Errorf("failed to get metrics enabled: %w", err)
		}
	}

	if cmd.Flags().Changed(HTTPMetricsIPV4HostKey) {
		config.HTTP.Metrics.IPV4Host, err = cmd.Flags().GetString(HTTPMetricsIPV4HostKey)
		if err != nil {
			return fmt.Errorf("failed to get metrics IPv4 host: %w", err)
		}
	}

	if cmd.Flags().Changed(HTTPMetricsIPV6HostKey) {
		config.HTTP.Metrics.IPV6Host, err = cmd.Flags().GetString(HTTPMetricsIPV6HostKey)
		if err != nil {
			return fmt.Errorf("failed to get metrics IPv6 host: %w", err)
		}
	}

	if cmd.Flags().Changed(HTTPMetricsPortKey) {
		config.HTTP.Metrics.Port, err = cmd.Flags().GetUint16(HTTPMetricsPortKey)
		if err != nil {
			return fmt.Errorf("failed to get metrics port: %w", err)
		}
	}

	if cmd.Flags().Changed(HTTPTracingEnabledKey) {
		config.HTTP.Tracing.Enabled, err = cmd.Flags().GetBool(HTTPTracingEnabledKey)
		if err != nil {
			return fmt.Errorf("failed to get tracing enabled: %w", err)
		}
	}

	if cmd.Flags().Changed(HTTPTracingOTLPEndKey) {
		config.HTTP.Tracing.OTLPEndpoint, err = cmd.Flags().GetString(HTTPTracingOTLPEndKey)
		if err != nil {
			return fmt.Errorf("failed to get tracing OTLP endpoint: %w", err)
		}
	}

	if cmd.Flags().Changed(HTTPCORSHostsKey) {
		config.HTTP.CORSHosts, err = cmd.Flags().GetStringSlice(HTTPCORSHostsKey)
		if err != nil {
			return fmt.Errorf("failed to get CORS hosts: %w", err)
		}
	}

	if cmd.Flags().Changed(HTTPBackendURLKey) {
		config.HTTP.BackendURL, err = cmd.Flags().GetString(HTTPBackendURLKey)
		if err != nil {
			return fmt.Errorf("failed to get backend URL: %w", err)
		}
	}

	if cmd.Flags().Changed(PersistenceDatabaseKey) {
		config.Persistence.Database, err = cmd.Flags().GetString(PersistenceDatabaseKey)
		if err != nil {
			return fmt.Errorf("failed to get database path: %w", err)
		}
	}

	if cmd.Flags().Changed(PersistenceUploadsKey) {
		config.Persistence.Uploads, err = cmd.Flags().GetString(PersistenceUploadsKey)
		if err != nil {
			return fmt.Errorf("failed to get uploads directory: %w", err)
		}
	}

	if cmd.Flags().Changed(RegistrationEnabledKey) {
		config.Registration.Enabled, err = cmd.Flags().GetBool(RegistrationEnabledKey)
		if err != nil {
			return fmt.Errorf("failed to get registration enabled: %w", err)
		}
	}

	if cmd.Flags().Changed(AuthGoogleClientIDKey) {
		config.Auth.Google.ClientID, err = cmd.Flags().GetString(AuthGoogleClientIDKey)
		if err != nil {
			return fmt.Errorf("failed to get Google OAuth client ID: %w", err)
		}
	}

	if cmd.Flags().Changed(AuthGoogleClientSecretKey) {
		config.Auth.Google.ClientSecret, err = cmd.Flags().GetString(AuthGoogleClientSecretKey)
		if err != nil {
			return fmt.Errorf("failed to get Google OAuth client secret: %w", err)
		}
	}

	if cmd.Flags().Changed(AuthGitHubClientIDKey) {
		config.Auth.GitHub.ClientID, err = cmd.Flags().GetString(AuthGitHubClientIDKey)
		if err != nil {
			return fmt.Errorf("failed to get GitHub OAuth client ID: %w", err)
		}
	}

	if cmd.Flags().Changed(AuthGitHubClientSecretKey) {
		config.Auth.GitHub.ClientSecret, err = cmd.Flags().GetString(AuthGitHubClientSecretKey)
		if err != nil {
			return fmt.Errorf("failed to get GitHub OAuth client secret: %w", err)
		}
	}

	if cmd.Flags().Changed(JWTSecretKey) {
		config.JWT.Secret, err = cmd.Flags().GetString(JWTSecretKey)
		if err != nil {
			return fmt.Errorf("failed to get JWT secret: %w", err)
		}
	}

	if cmd.Flags().Changed(MapboxPublicTokenKey) {
		config.Mapbox.PublicToken, err = cmd.Flags().GetString(MapboxPublicTokenKey)
		if err != nil {
			return fmt.Errorf("failed to get Mapbox public token: %w", err)
		}
	}

	if cmd.Flags().Changed(MapboxSecretTokenKey) {
		config.Mapbox.SecretToken, err = cmd.Flags().GetString(MapboxSecretTokenKey)
		if err != nil {
			return fmt.Errorf("failed to get Mapbox secret token: %w", err)
		}
	}

	return nil
}
