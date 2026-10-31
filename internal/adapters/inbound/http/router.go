// Package http adapts HTTP requests to application use cases.
package http

import (
	"errors"
	"log/slog"
	nethttp "net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/plusiv/go-api-boilerplate/internal/application/usecases"
	"github.com/plusiv/go-api-boilerplate/internal/domain/apperrors"
	"github.com/plusiv/go-api-boilerplate/internal/domain/entities"
	"github.com/plusiv/go-api-boilerplate/internal/domain/repositories"
)

type itemResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type errorResponse struct {
	Error  string   `json:"error"`
	Fields []string `json:"fields,omitempty"`
}

func response(i entities.Item) itemResponse { return itemResponse{ID: i.ID, Name: i.Name} }

// NewRouter registers the API routes and common middleware.
func NewRouter(items *usecases.Items, log *slog.Logger, cfg RouterConfig) (*echo.Echo, error) {
	if cfg.PprofEnabled && strings.TrimSpace(cfg.PprofToken) == "" {
		return nil, errors.New("profiling token is required")
	}
	if cfg.MetricsEnabled && strings.TrimSpace(cfg.MetricsToken) == "" {
		return nil, errors.New("metrics token is required")
	}
	e := echo.New()
	binder, err := newValidateBinder()
	if err != nil {
		return nil, err
	}
	e.Binder = binder
	var metricsMiddleware echo.MiddlewareFunc
	if cfg.MetricsEnabled {
		metricsMiddleware, err = registerMetrics(e, cfg.MetricsToken)
		if err != nil {
			return nil, err
		}
	}
	configureMiddleware(e, log, cfg, metricsMiddleware)
	e.HTTPErrorHandler = handleError
	if cfg.PprofEnabled {
		registerPprof(e, cfg.PprofToken)
	}
	e.GET("/healthz", func(c *echo.Context) error {
		return c.JSON(nethttp.StatusOK, struct {
			Status string `json:"status"`
		}{Status: "ok"})
	})
	e.POST("/items", func(c *echo.Context) error {
		var req struct {
			ID   string `json:"id" validate:"required,notblank,max=100"`
			Name string `json:"name" validate:"required,notblank,max=200"`
		}
		if err := c.Bind(&req); err != nil {
			return err
		}
		item, err := items.Create(c.Request().Context(), req.ID, req.Name)
		if err != nil {
			return err
		}
		return c.JSON(nethttp.StatusCreated, response(item))
	})
	e.GET("/items/:id", func(c *echo.Context) error {
		item, err := items.Get(c.Request().Context(), c.Param("id"))
		if err != nil {
			return err
		}
		return c.JSON(nethttp.StatusOK, response(item))
	})
	e.GET("/items", func(c *echo.Context) error {
		page := repositories.Page{Limit: 20}
		for key, dest := range map[string]*int{"limit": &page.Limit, "offset": &page.Offset} {
			if val := c.QueryParam(key); val != "" {
				n, err := strconv.Atoi(val)
				if err != nil {
					return apperrors.NewValidation("invalid input")
				}
				*dest = n
			}
		}
		var filters repositories.ItemFilters
		if name := c.QueryParam("name"); name != "" {
			filters.Name = &repositories.Filter[string]{Is: &name}
		}
		result, err := items.List(c.Request().Context(), filters, page)
		if err != nil {
			return err
		}
		out := make([]itemResponse, 0, len(result))
		for _, item := range result {
			out = append(out, response(item))
		}
		return c.JSON(nethttp.StatusOK, out)
	})
	return e, nil
}
