// Package config loads process configuration from environment variables.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Env      string `env:"APP_ENV" envDefault:"development"`
	HTTPAddr string `env:"HTTP_ADDR" envDefault:":8080"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	// DatabaseURL connects as the application role (member of app_rw, RLS enforced).
	DatabaseURL string `env:"DATABASE_URL,required"`
	// MigrateDatabaseURL connects as the schema owner. Only used by cmd/migrate.
	MigrateDatabaseURL string `env:"MIGRATE_DATABASE_URL"`

	// RootDomain hosts schools at {subdomain}.RootDomain. AppHost serves the creator dashboard,
	// AdminHost the platform admin. Hosts are compared without port.
	RootDomain string `env:"ROOT_DOMAIN" envDefault:"localhost"`
	AppHost    string `env:"APP_HOST" envDefault:"app.localhost"`
	AdminHost  string `env:"ADMIN_HOST" envDefault:"admin.localhost"`

	// TrustProxy makes the API read X-Forwarded-Host / X-Forwarded-For. Enable only behind the edge proxy.
	TrustProxy   bool          `env:"TRUST_PROXY" envDefault:"true"`
	CookieSecure bool          `env:"COOKIE_SECURE" envDefault:"false"`
	SessionTTL   time.Duration `env:"SESSION_TTL" envDefault:"720h"`
}

func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	cfg.RootDomain = strings.ToLower(cfg.RootDomain)
	cfg.AppHost = strings.ToLower(cfg.AppHost)
	cfg.AdminHost = strings.ToLower(cfg.AdminHost)
	if cfg.MigrateDatabaseURL == "" {
		cfg.MigrateDatabaseURL = cfg.DatabaseURL
	}
	return cfg, nil
}

func (c Config) IsProduction() bool { return c.Env == "production" }
