package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func clearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConfig(t *testing.T) {
	for _, tt := range []struct {
		name, addr, timeout, origins string
		wantAddr                     string
		wantTimeout                  time.Duration
		wantOrigins                  []string
		wantError                    bool
	}{
		{name: "defaults", wantAddr: ":8080", wantTimeout: 60 * time.Second, wantOrigins: []string{}},
		{name: "overrides", addr: "127.0.0.1:9000", timeout: "45s", origins: "https://one.example, https://two.example , ", wantAddr: "127.0.0.1:9000", wantTimeout: 45 * time.Second, wantOrigins: []string{"https://one.example", "https://two.example"}},
		{name: "bad duration", timeout: "bad", wantError: true},
		{name: "zero duration", timeout: "0s", wantError: true},
		{name: "negative duration", timeout: "-1s", wantError: true},
		{name: "blank address", addr: " ", wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t, "HTTP_ADDR", "HTTP_REQUEST_TIMEOUT", "CORS_ALLOWED_ORIGINS", "PPROF_ENABLED", "PPROF_TOKEN", "METRICS_ENABLED", "METRICS_TOKEN")
			if tt.addr != "" {
				t.Setenv("HTTP_ADDR", tt.addr)
			}
			if tt.timeout != "" {
				t.Setenv("HTTP_REQUEST_TIMEOUT", tt.timeout)
			}
			if tt.origins != "" {
				t.Setenv("CORS_ALLOWED_ORIGINS", tt.origins)
			}
			cfg, err := Load[Config](filepath.Join(t.TempDir(), "absent.env"))
			if err == nil {
				err = cfg.Validate()
			}
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Address != tt.wantAddr || cfg.RequestTimeout != tt.wantTimeout || !reflect.DeepEqual(cfg.AllowedOrigins, tt.wantOrigins) {
				t.Fatalf("unexpected config: %+v", cfg)
			}
		})
	}
}

func TestDotenv(t *testing.T) {
	clearEnv(t, "HTTP_ADDR", "HTTP_REQUEST_TIMEOUT", "CORS_ALLOWED_ORIGINS", "PPROF_ENABLED", "PPROF_TOKEN", "METRICS_ENABLED", "METRICS_TOKEN")
	path := filepath.Join(t.TempDir(), "settings.env")
	if err := os.WriteFile(path, []byte("HTTP_ADDR=:8001\nHTTP_REQUEST_TIMEOUT=30s\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HTTP_ADDR", ":9001")
	cfg, err := Load[Config](path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Address != ":9001" || cfg.RequestTimeout != 30*time.Second {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if err := os.WriteFile(path, []byte("BROKEN='unterminated\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load[Config](path); err == nil || !strings.Contains(err.Error(), "dotenv") {
		t.Fatalf("expected dotenv parse error, got %v", err)
	}
}

func TestProfilingConfig(t *testing.T) {
	for _, tt := range []struct {
		name, enabled, token string
		wantError            bool
	}{
		{name: "disabled"}, {name: "enabled without token", enabled: "true", wantError: true},
		{name: "blank token", enabled: "true", token: " ", wantError: true}, {name: "enabled", enabled: "true", token: "secret"},
		{name: "invalid boolean", enabled: "invalid", wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t, "HTTP_ADDR", "HTTP_REQUEST_TIMEOUT", "CORS_ALLOWED_ORIGINS", "PPROF_ENABLED", "PPROF_TOKEN", "METRICS_ENABLED", "METRICS_TOKEN")
			if tt.enabled != "" {
				t.Setenv("PPROF_ENABLED", tt.enabled)
			}
			if tt.token != "" {
				t.Setenv("PPROF_TOKEN", tt.token)
			}
			cfg, err := Load[Config](filepath.Join(t.TempDir(), "absent.env"))
			if err == nil {
				err = cfg.Validate()
			}
			if (err != nil) != tt.wantError {
				t.Fatalf("error: %v", err)
			}
		})
	}
}

func TestMetricsConfig(t *testing.T) {
	for _, tt := range []struct {
		name, enabled, token string
		wantError            bool
	}{
		{name: "disabled"}, {name: "missing token", enabled: "true", wantError: true},
		{name: "blank token", enabled: "true", token: " ", wantError: true}, {name: "enabled", enabled: "true", token: "secret"},
		{name: "invalid enabled", enabled: "invalid", wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t, "HTTP_ADDR", "HTTP_REQUEST_TIMEOUT", "CORS_ALLOWED_ORIGINS", "PPROF_ENABLED", "PPROF_TOKEN", "METRICS_ENABLED", "METRICS_TOKEN")
			if tt.enabled != "" {
				t.Setenv("METRICS_ENABLED", tt.enabled)
			}
			if tt.token != "" {
				t.Setenv("METRICS_TOKEN", tt.token)
			}
			cfg, err := Load[Config](filepath.Join(t.TempDir(), "absent.env"))
			if err == nil {
				err = cfg.Validate()
			}
			if (err != nil) != tt.wantError {
				t.Fatalf("error: %v", err)
			}
		})
	}
}
