package config

//go:generate go tool configulator -type Config

import (
	"strings"

	configulator "github.com/USA-RedDragon/configulator/v2"
	cpflag "github.com/USA-RedDragon/configulator/v2/flags/pflag"
	"github.com/go-errors/errors"
	"github.com/goccy/go-yaml"
	"github.com/spf13/pflag"
)

type Config struct {
	HTTP               HTTP         `name:"http"`
	Persistence        Persistence  `name:"persistence"`
	Registration       Registration `name:"registration"`
	Auth               Auth         `name:"auth"`
	JWT                JWT          `name:"jwt"`
	Mapbox             Mapbox       `name:"mapbox"`
	NATS               NATS         `name:"nats"`
	ParallelLogParsers uint         `name:"parallel_log_parsers" default:"4" description:"Number of parallel log parsers"`
	LogLevel           LogLevel     `name:"log_level" default:"info" description:"Log level, one of: debug, info, warn, error"`
}

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type NATS struct {
	Enabled bool   `name:"enabled" description:"Enable NATS. Required when running more than one instance, such as blue/green deployments, so websockets reach the instance the device is connected to"`
	URL     string `name:"url" description:"NATS URL"`
	Token   string `name:"token" secret:"true" description:"NATS token"`
}

type JWT struct {
	Secret string `name:"secret" required:"true" secret:"true" description:"JWT signing secret"`
}

type Auth struct {
	Google Google `name:"google"`
	GitHub GitHub `name:"github"`
	Custom Custom `name:"custom"`
}

type Mapbox struct {
	SecretToken string `name:"secret_token" required:"true" secret:"true" description:"Mapbox secret token"`
	PublicToken string `name:"public_token" required:"true" description:"Mapbox public token"`
}

type Google struct {
	Enabled      bool   `name:"enabled" description:"Enable Google OAuth"`
	ClientID     string `name:"client_id" description:"Google OAuth client ID"`
	ClientSecret string `name:"client_secret" secret:"true" description:"Google OAuth client secret"`
}

type GitHub struct {
	Enabled      bool   `name:"enabled" description:"Enable GitHub OAuth"`
	ClientID     string `name:"client_id" description:"GitHub OAuth client ID"`
	ClientSecret string `name:"client_secret" secret:"true" description:"GitHub OAuth client secret"`
}

type Custom struct {
	Enabled      bool   `name:"enabled" description:"Enable custom OAuth"`
	ClientID     string `name:"client_id" description:"Custom OAuth client ID"`
	ClientSecret string `name:"client_secret" secret:"true" description:"Custom OAuth client secret"`
	TokenURL     string `name:"token_url" description:"Custom OAuth token URL"`
	UserURL      string `name:"user_url" description:"Custom OAuth user URL"`
}

type Registration struct {
	Enabled bool `name:"enabled" description:"Enable user registration"`
}

type Persistence struct {
	Database Database `name:"database"`
	Uploads  Uploads  `name:"uploads"`
}

type UploadsDriver string

const (
	UploadsDriverFilesystem UploadsDriver = "filesystem"
	UploadsDriverS3         UploadsDriver = "s3"
)

type Uploads struct {
	Driver            UploadsDriver     `name:"driver" default:"filesystem" description:"Storage driver for uploaded videos and driving logs, one of: filesystem, s3"`
	FilesystemOptions FilesystemOptions `name:"filesystem_options"`
	S3Options         S3Options         `name:"s3_options"`
}

type FilesystemOptions struct {
	Directory string `name:"directory" default:"uploads/" description:"Filesystem uploads directory, created if it does not exist"`
}

type S3Options struct {
	Region   string `name:"region" description:"S3 region. Credentials come from the standard AWS environment variables, the AWS CLI config or an IAM role"`
	Bucket   string `name:"bucket" description:"S3 bucket"`
	Endpoint string `name:"endpoint" description:"Custom S3 endpoint, which switches to path-style addressing"`
}

type DatabaseDriver string

const (
	DatabaseDriverSQLite   DatabaseDriver = "sqlite"
	DatabaseDriverMySQL    DatabaseDriver = "mysql"
	DatabaseDriverPostgres DatabaseDriver = "postgres"
)

type Database struct {
	Driver          DatabaseDriver `name:"driver" default:"sqlite" description:"Database driver, one of: sqlite, mysql, postgres"`
	Database        string         `name:"database" default:"rtz.db" description:"Path to the SQLite database file, or the database name for other drivers"`
	Username        string         `name:"username" description:"Database username"`
	Password        string         `name:"password" secret:"true" description:"Database password"`
	Host            string         `name:"host" description:"Database host, required for mysql and postgres"`
	Port            uint16         `name:"port" description:"Database port, 0 uses the driver's default"`
	ExtraParameters string         `name:"extra_parameters" description:"Extra parameters passed to the database driver"`
}

type Tracing struct {
	Enabled      bool   `name:"enabled" description:"Enable OpenTelemetry tracing"`
	OTLPEndpoint string `name:"otlp_endpoint" description:"OpenTelemetry collector endpoint, required when tracing is enabled"`
}

type PProf struct {
	Enabled bool `name:"enabled" description:"Enable pprof"`
}

type Metrics struct {
	IPV4Host string `name:"ipv4_host" default:"127.0.0.1" description:"Prometheus metrics server IPv4 host"`
	IPV6Host string `name:"ipv6_host" default:"::1" description:"Prometheus metrics server IPv6 host"`
	Port     uint16 `name:"port" default:"8081" description:"Prometheus metrics server port, shared by IPv4 and IPv6"`
	Enabled  bool   `name:"enabled" description:"Enable the Prometheus metrics server"`
}

type HTTP struct {
	IPV4Host       string   `name:"ipv4_host" default:"0.0.0.0" description:"HTTP server IPv4 host"`
	IPV6Host       string   `name:"ipv6_host" default:"::" description:"HTTP server IPv6 host"`
	Port           uint16   `name:"port" default:"8080" description:"HTTP server port, shared by IPv4 and IPv6"`
	Tracing        Tracing  `name:"tracing"`
	BackendURL     string   `name:"backend_url" required:"true" description:"Public URL of this server"`
	PProf          PProf    `name:"pprof"`
	TrustedProxies []string `name:"trusted_proxies" description:"IP addresses or CIDR ranges of reverse proxies trusted to set X-Forwarded-For"`
	Metrics        Metrics  `name:"metrics"`
	CORSHosts      []string `name:"cors_hosts" description:"Hosts allowed by CORS"`
}

// NewConfigulator registers the config flags on fs and returns the loader
// that reads defaults, config.yaml (or --config), environment variables and
// those flags.
func NewConfigulator(fs *pflag.FlagSet) *configulator.Configulator[Config] {
	c := configulator.New(ConfigSchema()).
		WithEnvironmentVariables(&configulator.EnvironmentVariableOptions{Separator: "__"}).
		WithFile(&configulator.FileOptions{
			Search:   []string{"config.yaml"},
			Decoders: configulator.Decoders{".yaml": yaml.Unmarshal, ".yml": yaml.Unmarshal},
		})
	return cpflag.Bind(c, fs, ConfigPFlagHooks(), nil)
}

var (
	ErrJWTSecretRequired          = errors.New("JWT secret is required")
	ErrBackendURLRequired         = errors.New("Backend URL is required")
	ErrOTLPEndpointRequired       = errors.New("OTLP endpoint is required when tracing is enabled")
	ErrMapboxPublicTokenRequired  = errors.New("Mapbox public token is required")
	ErrMapboxSecretTokenRequired  = errors.New("Mapbox secret token is required")
	ErrDBHostRequired             = errors.New("Database host is required")
	ErrDBDatabaseRequired         = errors.New("Database name is required")
	ErrDatabaseDriverRequired     = errors.New("Database driver is required")
	ErrNATSURLRequired            = errors.New("NATS URL is required")
	ErrGitHubOAuthRequired        = errors.New("GitHub OAuth client ID and secret are required")
	ErrGoogleOAuthRequired        = errors.New("Google OAuth client ID and secret are required")
	ErrCustomOAuthRequired        = errors.New("Custom OAuth client ID and secret are required")
	ErrCustomTokenURLRequired     = errors.New("Custom OAuth token URL is required")
	ErrCustomUserURLRequired      = errors.New("Custom OAuth user URL is required")
	ErrParallelLogParsersNotZero  = errors.New("Number of parallel log parsers must be greater than zero")
	ErrInvalidLogLevel            = errors.New("Invalid log level")
	ErrUploadsFSDirectoryRequired = errors.New("Filesystem uploads directory is required")
	ErrUploadsS3BucketRequired    = errors.New("S3 bucket is required")
	ErrUploadsS3RegionRequired    = errors.New("S3 region is required")
	ErrInvalidDatabaseDriver      = errors.New("Invalid database driver, must be one of: sqlite, mysql, postgres")
	ErrInvalidUploadsDriver       = errors.New("Invalid uploads driver, must be one of: filesystem, s3")
)

// Validate lowercases the driver names, then checks the settings that
// depend on each other.
func (c *Config) Validate() error {
	c.Persistence.Database.Driver = DatabaseDriver(strings.ToLower(string(c.Persistence.Database.Driver)))
	c.Persistence.Uploads.Driver = UploadsDriver(strings.ToLower(string(c.Persistence.Uploads.Driver)))

	if c.JWT.Secret == "" {
		return ErrJWTSecretRequired
	}
	if c.HTTP.BackendURL == "" {
		return ErrBackendURLRequired
	}
	if c.HTTP.Tracing.Enabled && c.HTTP.Tracing.OTLPEndpoint == "" {
		return ErrOTLPEndpointRequired
	}
	if c.Mapbox.PublicToken == "" {
		return ErrMapboxPublicTokenRequired
	}
	if c.Mapbox.SecretToken == "" {
		return ErrMapboxSecretTokenRequired
	}
	if err := c.Persistence.validate(); err != nil {
		return err
	}
	if c.NATS.Enabled && c.NATS.URL == "" {
		return ErrNATSURLRequired
	}
	if err := c.Auth.validate(); err != nil {
		return err
	}
	if c.ParallelLogParsers == 0 {
		return ErrParallelLogParsersNotZero
	}
	switch c.LogLevel {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		return ErrInvalidLogLevel
	}

	return nil
}

func (p *Persistence) validate() error {
	if p.Database.Driver == "" {
		return ErrDatabaseDriverRequired
	}
	switch p.Database.Driver {
	case DatabaseDriverSQLite, DatabaseDriverMySQL, DatabaseDriverPostgres:
	default:
		return ErrInvalidDatabaseDriver
	}
	if p.Database.Driver != DatabaseDriverSQLite && p.Database.Host == "" {
		return ErrDBHostRequired
	}
	if p.Database.Database == "" {
		return ErrDBDatabaseRequired
	}
	switch p.Uploads.Driver {
	case UploadsDriverFilesystem, UploadsDriverS3:
	default:
		return ErrInvalidUploadsDriver
	}
	if p.Uploads.Driver == UploadsDriverFilesystem && p.Uploads.FilesystemOptions.Directory == "" {
		return ErrUploadsFSDirectoryRequired
	}
	if p.Uploads.Driver == UploadsDriverS3 && p.Uploads.S3Options.Bucket == "" {
		return ErrUploadsS3BucketRequired
	}
	if p.Uploads.Driver == UploadsDriverS3 && p.Uploads.S3Options.Region == "" {
		return ErrUploadsS3RegionRequired
	}
	return nil
}

func (a *Auth) validate() error {
	if a.GitHub.Enabled && (a.GitHub.ClientID == "" || a.GitHub.ClientSecret == "") {
		return ErrGitHubOAuthRequired
	}
	if a.Google.Enabled && (a.Google.ClientID == "" || a.Google.ClientSecret == "") {
		return ErrGoogleOAuthRequired
	}
	if a.Custom.Enabled && (a.Custom.ClientID == "" || a.Custom.ClientSecret == "") {
		return ErrCustomOAuthRequired
	}
	if a.Custom.Enabled && a.Custom.TokenURL == "" {
		return ErrCustomTokenURLRequired
	}
	if a.Custom.Enabled && a.Custom.UserURL == "" {
		return ErrCustomUserURLRequired
	}
	return nil
}
