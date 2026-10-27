package config

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func clearEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"PORT", "SERVICE_NAME", "LOG_LEVEL", "LOG_DIR",
		"DATABASE_URL", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"DB_SSLMODE", "DB_MAX_CONNS", "DB_CONNECT_TIMEOUT", "DB_AUTO_MIGRATE",
		"JWT_SECRET", "JWT_ISSUER", "JWT_AUDIENCE", "JWT_TTL",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadUsesDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ServiceName != "boilerplate" || cfg.Port != "1111" {
		t.Fatalf("service = %q port = %q", cfg.ServiceName, cfg.Port)
	}
	if cfg.Database.Name != "boilerplate" {
		t.Fatalf("database name = %q, want boilerplate", cfg.Database.Name)
	}
	if cfg.Database.MaxConns != 10 || cfg.Database.ConnectTimeout != 5*time.Second {
		t.Fatalf("database pool config = %+v", cfg.Database)
	}
	if cfg.Database.AutoMigrate {
		t.Fatal("DB_AUTO_MIGRATE must default to false")
	}
	if cfg.Auth.JWTTTL != time.Hour {
		t.Fatalf("jwt ttl = %v, want 1h", cfg.Auth.JWTTTL)
	}
	if cfg.Auth.JWTIssuer != "boilerplate" || cfg.Auth.JWTAudience != "boilerplate-api" {
		t.Fatalf("jwt config = %+v", cfg.Auth)
	}
}

func TestLoadBuildsEncodedDatabaseURL(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("DB_USER", "app")
	t.Setenv("DB_PASSWORD", "p@ss word")
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("DB_NAME", "service")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	parsed, err := url.Parse(cfg.Database.URL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	password, _ := parsed.User.Password()
	if parsed.User.Username() != "app" || password != "p@ss word" {
		t.Fatalf("credentials = %q / %q", parsed.User.Username(), password)
	}
	if parsed.Host != "db.internal:6543" || parsed.Path != "/service" {
		t.Fatalf("database URL = %q", cfg.Database.URL)
	}
	if parsed.Query().Get("sslmode") != "disable" {
		t.Fatalf("sslmode = %q, want disable", parsed.Query().Get("sslmode"))
	}
}

func TestLoadHonorsDatabaseURLOverride(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("DATABASE_URL", "postgres://user:pass@remote:5432/from-url?sslmode=require")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.Name != "from-url" {
		t.Fatalf("database name = %q, want from-url", cfg.Database.Name)
	}
	if cfg.Database.URL != "postgres://user:pass@remote:5432/from-url?sslmode=require" {
		t.Fatalf("database URL = %q", cfg.Database.URL)
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		secret  string
		wantErr string
	}{
		{name: "port not a number", key: "PORT", value: "abc", secret: strings.Repeat("s", 32), wantErr: "PORT"},
		{name: "port out of range", key: "PORT", value: "70000", secret: strings.Repeat("s", 32), wantErr: "PORT"},
		{name: "short jwt secret", secret: "too-short", wantErr: "JWT_SECRET"},
		{name: "invalid jwt ttl", key: "JWT_TTL", value: "soon", secret: strings.Repeat("s", 32), wantErr: "JWT_TTL"},
		{name: "zero max conns", key: "DB_MAX_CONNS", value: "0", secret: strings.Repeat("s", 32), wantErr: "DB_MAX_CONNS"},
		{name: "invalid connect timeout", key: "DB_CONNECT_TIMEOUT", value: "later", secret: strings.Repeat("s", 32), wantErr: "DB_CONNECT_TIMEOUT"},
		{name: "invalid auto migrate", key: "DB_AUTO_MIGRATE", value: "maybe", secret: strings.Repeat("s", 32), wantErr: "DB_AUTO_MIGRATE"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("JWT_SECRET", test.secret)
			if test.key != "" {
				t.Setenv(test.key, test.value)
			}

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Load() error = %v, want containing %q", err, test.wantErr)
			}
		})
	}
}

func TestLoadDatabaseDoesNotRequireJWTSecret(t *testing.T) {
	clearEnv(t)

	database, err := LoadDatabase()
	if err != nil {
		t.Fatalf("LoadDatabase() error = %v", err)
	}
	if database.Name != "boilerplate" {
		t.Fatalf("database name = %q", database.Name)
	}
}
