package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/plusiv/go-api-boilerplate/internal/application/usecases"
	"github.com/plusiv/go-api-boilerplate/internal/domain/entities"
	"github.com/plusiv/go-api-boilerplate/internal/domain/repositories"
	"github.com/plusiv/go-api-boilerplate/internal/infrastructure/logger"
)

type loggingRepo struct{}

func (loggingRepo) Create(ctx context.Context, _ entities.Item) error {
	logger.FromContext(ctx).InfoContext(ctx, "repository called")
	return nil
}
func (loggingRepo) Get(context.Context, string) (entities.Item, error) {
	return entities.Item{}, errors.New("private database error")
}
func (loggingRepo) List(context.Context, repositories.ItemFilters, repositories.Page) ([]entities.Item, error) {
	return nil, nil
}

func TestRequestMiddleware(t *testing.T) {
	var logs bytes.Buffer
	router, err := NewRouter(usecases.NewItems(loggingRepo{}), slog.New(slog.NewJSONHandler(&logs, nil)), RouterConfig{CORSOrigins: []string{"https://example.com"}, RequestTimeout: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/timeout", func(c *echo.Context) error { <-c.Request().Context().Done(); return c.Request().Context().Err() })
	router.GET("/panic", func(_ *echo.Context) error { panic("test panic") })
	tests := []struct {
		name, method, path, body, id, origin string
		status                               int
		contains                             string
	}{
		{"valid request", nethttp.MethodPost, "/items", `{"id":"1","name":"test"}`, "known-id", "", 201, `"id":"1"`},
		{"validation", nethttp.MethodPost, "/items", `{"id":" ","name":""}`, "validation-id", "", 422, `"id: notblank"`},
		{"malformed", nethttp.MethodPost, "/items", `{`, "", "", 400, `"Bad Request"`},
		{"too large", nethttp.MethodPost, "/items", strings.Repeat("a", (512<<10)+1), "", "", 413, `"Request Entity Too Large"`},
		{"internal error", nethttp.MethodGet, "/items/1", "", "error-id", "", 500, `"internal server error"`},
		{"timeout", nethttp.MethodGet, "/timeout", "", "", "", 503, `"Service Unavailable"`},
		{"panic", nethttp.MethodGet, "/panic", "", "", "", 500, `"internal server error"`},
		{"cors", nethttp.MethodGet, "/healthz", "", "", "https://example.com", 200, `"ok"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs.Reset()
			req := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.id != "" {
				req.Header.Set(echo.HeaderXRequestID, tt.id)
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.contains) {
				t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
			}
			id := rec.Header().Get(echo.HeaderXRequestID)
			if id == "" || (tt.id != "" && id != tt.id) {
				t.Fatalf("request id %q", id)
			}
			if tt.origin != "" && rec.Header().Get("Access-Control-Allow-Origin") != tt.origin {
				t.Fatalf("CORS headers: %v", rec.Header())
			}
			var found bool
			for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
				var entry map[string]any
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					t.Fatal(err)
				}
				if entry["msg"] == "http request" {
					found = true
					if entry["request_id"] != id || entry["status"] != float64(tt.status) {
						t.Fatalf("access log: %v", entry)
					}
				}
			}
			if !found {
				t.Fatal("missing access log")
			}
			if tt.name == "valid request" && !strings.Contains(logs.String(), `"msg":"repository called"`) {
				t.Fatal("missing repository log")
			}
			if tt.name == "valid request" && !strings.Contains(logs.String(), `"request_id":"known-id"`) {
				t.Fatal("missing propagated request id")
			}
			if tt.name == "internal error" && strings.Contains(rec.Body.String(), "private database error") {
				t.Fatal("internal error leaked")
			}
		})
	}
}

func TestStreamingBodyLimit(t *testing.T) {
	router, err := NewRouter(usecases.NewItems(loggingRepo{}), slog.New(slog.NewTextHandler(io.Discard, nil)), RouterConfig{})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(context.Background(), nethttp.MethodPost, "/items", strings.NewReader(strings.Repeat("x", (512<<10)+1)))
	req.ContentLength = -1
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != nethttp.StatusRequestEntityTooLarge {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}
