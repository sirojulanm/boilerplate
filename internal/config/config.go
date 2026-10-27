package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/helper"
)

// Config contains every runtime setting the service needs. Credentials are
// read from the environment and are never written to logs.
type Config struct {
	ServiceName string
	Port        string
	LogLevel    string
	LogDir      string
	Database    Database
	Auth        Auth
}

// Database controls the PostgreSQL connection and optional startup migration.
type Database struct {
	URL            string
	Name           string
	MaxConns       int32
	ConnectTimeout time.Duration
	AutoMigrate    bool
}

// Auth controls JWT signing and validation.
type Auth struct {
	JWTSecret   string
	JWTIssuer   string
	JWTAudience string
	JWTTTL      time.Duration
}

// Load reads and validates the full application configuration.
func Load() (Config, error) {
	database, err := LoadDatabase()
	if err != nil {
		return Config{}, err
	}

	port := helper.Env("PORT", constant.DefaultPort)
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return Config{}, fmt.Errorf("PORT must be a number between 1 and 65535, got %q", port)
	}

	ttl, err := time.ParseDuration(helper.Env("JWT_TTL", constant.DefaultJWTTTL))
	if err != nil || ttl <= 0 {
		return Config{}, fmt.Errorf("JWT_TTL must be a positive duration, got %q", helper.Env("JWT_TTL", constant.DefaultJWTTTL))
	}

	secret := os.Getenv("JWT_SECRET")
	if len(secret) < constant.MinJWTSecretLength {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least %d characters", constant.MinJWTSecretLength)
	}

	return Config{
		ServiceName: helper.Env("SERVICE_NAME", constant.DefaultServiceName),
		Port:        port,
		LogLevel:    helper.Env("LOG_LEVEL", constant.DefaultLogLevel),
		LogDir:      helper.Env("LOG_DIR", constant.DefaultLogDir),
		Database:    database,
		Auth: Auth{
			JWTSecret:   secret,
			JWTIssuer:   helper.Env("JWT_ISSUER", constant.DefaultJWTIssuer),
			JWTAudience: helper.Env("JWT_AUDIENCE", constant.DefaultJWTAudience),
			JWTTTL:      ttl,
		},
	}, nil
}

// LoadDatabase reads the PostgreSQL settings only. Migration commands use this
// so they do not require JWT configuration.
func LoadDatabase() (Database, error) {
	name := helper.Env("DB_NAME", constant.DefaultDBName)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = buildDatabaseURL(name)
	} else if parsedName := databaseNameFromURL(dsn); parsedName != "" {
		name = parsedName
	}

	maxConns, err := strconv.ParseInt(helper.Env("DB_MAX_CONNS", constant.DefaultDBMaxConns), 10, 32)
	if err != nil || maxConns < 1 {
		return Database{}, fmt.Errorf("DB_MAX_CONNS must be a positive integer, got %q", helper.Env("DB_MAX_CONNS", constant.DefaultDBMaxConns))
	}

	connectTimeout, err := time.ParseDuration(helper.Env("DB_CONNECT_TIMEOUT", constant.DefaultDBConnectTimeout))
	if err != nil || connectTimeout <= 0 {
		return Database{}, fmt.Errorf("DB_CONNECT_TIMEOUT must be a positive duration, got %q", helper.Env("DB_CONNECT_TIMEOUT", constant.DefaultDBConnectTimeout))
	}

	autoMigrate, err := strconv.ParseBool(helper.Env("DB_AUTO_MIGRATE", constant.DefaultDBAutoMigrate))
	if err != nil {
		return Database{}, fmt.Errorf("DB_AUTO_MIGRATE must be true or false, got %q", helper.Env("DB_AUTO_MIGRATE", constant.DefaultDBAutoMigrate))
	}

	return Database{
		URL:            dsn,
		Name:           name,
		MaxConns:       int32(maxConns),
		ConnectTimeout: connectTimeout,
		AutoMigrate:    autoMigrate,
	}, nil
}

func buildDatabaseURL(name string) string {
	host := net.JoinHostPort(
		helper.Env("DB_HOST", constant.DefaultDBHost),
		helper.Env("DB_PORT", constant.DefaultDBPort),
	)

	dsn := &url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			helper.Env("DB_USER", constant.DefaultDBUser),
			helper.Env("DB_PASSWORD", constant.DefaultDBPassword),
		),
		Host: host,
		Path: "/" + name,
	}

	query := dsn.Query()
	query.Set("sslmode", helper.Env("DB_SSLMODE", constant.DefaultDBSSLMode))
	dsn.RawQuery = query.Encode()

	return dsn.String()
}

func databaseNameFromURL(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Path, "/")
}
