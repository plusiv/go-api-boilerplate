package http

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/plusiv/go-api-boilerplate/internal/application/usecases"
	"github.com/plusiv/go-api-boilerplate/internal/domain/apperrors"
	"github.com/rotisserie/eris"
)

func TestErrorMapping(t *testing.T) {
	private := errors.New("private connection details")
	for _, tt := range []struct {
		name    string
		err     error
		status  int
		message string
		logged  bool
	}{
		{"not found", eris.Wrap(apperrors.NewNotFound("item"), "lookup"), 404, "item not found", false},
		{"conflict", apperrors.NewConflict("already exists"), 409, "already exists", false},
		{"validation", apperrors.NewValidation("invalid input"), 422, "invalid input", false},
		{"business", apperrors.NewBusinessRule("not allowed"), 422, "not allowed", false},
		{"rate", apperrors.NewRateLimit("slow down"), 429, "slow down", false},
		{"gone", apperrors.NewGone("removed"), 410, "removed", false},
		{"internal", apperrors.NewInternal(private), 500, "internal server error", true},
		{"unknown", private, 500, "internal server error", true},
		{"internal wrapping http error", apperrors.NewInternal(echo.ErrStatusRequestEntityTooLarge), 500, "internal server error", true},
		{"unknown kind", apperrors.New("unexpected", "private connection details"), 500, "internal server error", true},
		{"echo error", echo.NewHTTPError(400, "private connection details"), 400, "Bad Request", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			router, err := NewRouter(usecases.NewItems(loggingRepo{}), slog.New(slog.NewJSONHandler(&logs, nil)), RouterConfig{})
			if err != nil {
				t.Fatal(err)
			}
			router.GET("/error", func(_ *echo.Context) error { return tt.err })
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), nethttp.MethodGet, "/error", nil))
			if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.message) || strings.Contains(rec.Body.String(), "private connection details") {
				t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
			}
			if tt.logged && !strings.Contains(logs.String(), "request failed") {
				t.Fatalf("missing cause in logs: %s", logs.String())
			}
		})
	}
}
