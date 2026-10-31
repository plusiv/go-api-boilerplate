package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/plusiv/go-api-boilerplate/internal/adapters/outbound/memory"
	"github.com/plusiv/go-api-boilerplate/internal/application/usecases"
)

func TestItemHTTP(t *testing.T) {
	router, err := NewRouter(usecases.NewItems(memory.NewItems()), slog.New(slog.NewTextHandler(io.Discard, nil)), RouterConfig{})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, method, path, body string
		status                   int
		contains                 string
	}{
		{"health", http.MethodGet, "/healthz", "", 200, `"ok"`},
		{"create", http.MethodPost, "/items", `{"id":"a","name":"Alpha"}`, 201, `"id":"a"`},
		{"duplicate", http.MethodPost, "/items", `{"id":"a","name":"Alpha"}`, 409, `"item already exists"`},
		{"invalid", http.MethodPost, "/items", `{"id":""}`, 422, `"invalid input"`},
		{"get", http.MethodGet, "/items/a", "", 200, `"Alpha"`},
		{"missing", http.MethodGet, "/items/missing", "", 404, `"item not found"`},
		{"filter", http.MethodGet, "/items?name=other", "", 200, `[]`},
		{"bad limit", http.MethodGet, "/items?limit=101", "", 422, `"invalid input"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.contains) {
				t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
			}
		})
	}
}
