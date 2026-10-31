package http

import (
	"context"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/plusiv/go-api-boilerplate/internal/adapters/outbound/memory"
	"github.com/plusiv/go-api-boilerplate/internal/application/usecases"
)

func TestPprofEndpoints(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tt := range []struct {
		name                       string
		enabled                    bool
		token, authorization, path string
		status                     int
		contains                   string
	}{
		{"disabled", false, "", "", "/metrics/pprof/", 404, ""},
		{"missing token", true, "secret", "", "/metrics/pprof/", 401, ""},
		{"wrong token", true, "secret", "Bearer wrong", "/metrics/pprof/heap", 401, ""},
		{"index", true, "secret", "Bearer secret", "/metrics/pprof/", 200, "heap"},
		{"redirect", true, "secret", "Bearer secret", "/metrics/pprof", 308, ""},
		{"heap", true, "secret", "Bearer secret", "/metrics/pprof/heap?debug=1", 200, "heap profile"},
		{"unknown profile", true, "secret", "Bearer secret", "/metrics/pprof/unknown", 404, ""},
		{"cmdline", true, "secret", "Bearer secret", "/metrics/pprof/cmdline", 200, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			router, err := NewRouter(usecases.NewItems(memory.NewItems()), log, RouterConfig{PprofEnabled: tt.enabled, PprofToken: tt.token, RequestTimeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequestWithContext(context.Background(), nethttp.MethodGet, tt.path, nil)
			req.Header.Set("Authorization", tt.authorization)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.contains) {
				t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
			}
		})
	}
	if _, err := NewRouter(usecases.NewItems(memory.NewItems()), log, RouterConfig{PprofEnabled: true}); err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestPprofProfileOutlastsRequestTimeout(t *testing.T) {
	router, err := NewRouter(usecases.NewItems(memory.NewItems()), slog.New(slog.NewTextHandler(io.Discard, nil)), RouterConfig{PprofEnabled: true, PprofToken: "secret", RequestTimeout: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(context.Background(), nethttp.MethodGet, "/metrics/pprof/profile?seconds=1", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != nethttp.StatusOK || rec.Body.Len() == 0 {
		t.Fatalf("status %d, body length %d", rec.Code, rec.Body.Len())
	}
}
