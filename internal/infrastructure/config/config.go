// Package config loads application settings from the environment and optional dotenv files.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config groups the settings needed to start the API.
type Config struct {
	HTTP
	CORS
	Profiling
	Metrics
}

// HTTP holds HTTP server settings.
type HTTP struct {
	Address        string        `envconfig:"HTTP_ADDR" default:":8080"`
	RequestTimeout time.Duration `envconfig:"HTTP_REQUEST_TIMEOUT" default:"60s"`
}

// CORS holds the allowed origins for cross-origin requests.
type CORS struct {
	AllowedOrigins []string `envconfig:"CORS_ALLOWED_ORIGINS"`
}

// Profiling controls the optional HTTP profiling endpoints.
type Profiling struct {
	Enabled bool   `envconfig:"PPROF_ENABLED" default:"false"`
	Token   string `envconfig:"PPROF_TOKEN"`
}

// Metrics controls the optional Prometheus scrape endpoint.
type Metrics struct {
	MetricsEnabled bool   `envconfig:"METRICS_ENABLED" default:"false"`
	MetricsToken   string `envconfig:"METRICS_TOKEN"`
}

// Load reads optional dotenv files, then decodes environment variables into T.
// With no filenames it reads .env. Existing environment variables take precedence.
func Load[T any](filenames ...string) (*T, error) {
	if err := godotenv.Load(filenames...); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("load dotenv file: %w", err)
	}
	cfg := new(T)
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("load environment: %w", err)
	}
	return cfg, nil
}

// Validate checks the server settings and normalizes allowed origins.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Address) == "" {
		return errors.New("HTTP_ADDR must not be empty")
	}
	if c.RequestTimeout <= 0 {
		return errors.New("HTTP_REQUEST_TIMEOUT must be a positive duration")
	}
	if c.Enabled && strings.TrimSpace(c.Token) == "" {
		return errors.New("PPROF_TOKEN is required when profiling is enabled")
	}
	if c.MetricsEnabled && strings.TrimSpace(c.MetricsToken) == "" {
		return errors.New("METRICS_TOKEN is required when metrics are enabled")
	}
	origins := make([]string, 0, len(c.AllowedOrigins))
	for _, origin := range c.AllowedOrigins {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	c.AllowedOrigins = origins
	return nil
}
