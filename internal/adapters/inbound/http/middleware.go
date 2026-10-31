package http

import (
	"log/slog"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/plusiv/go-api-boilerplate/internal/infrastructure/logger"
)

// RouterConfig controls request middleware. A nil CORSOrigins disables CORS.
type RouterConfig struct {
	CORSOrigins    []string
	RequestTimeout time.Duration
	PprofEnabled   bool
	PprofToken     string
	MetricsEnabled bool
	MetricsToken   string
}

func configureMiddleware(e *echo.Echo, log *slog.Logger, cfg RouterConfig, metrics echo.MiddlewareFunc) {
	// Limit untrusted IDs before adding them to logs or response headers.
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			id := c.Request().Header.Get(echo.HeaderXRequestID)
			if len(id) > 128 || strings.ContainsAny(id, " \t") {
				c.Request().Header.Del(echo.HeaderXRequestID)
			}
			return next(c)
		}
	})
	e.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{RequestIDHandler: func(c *echo.Context, id string) {
		ctx := logger.WithContext(c.Request().Context(), log.With("request_id", id))
		c.SetRequest(c.Request().WithContext(ctx))
	}}))
	if metrics != nil {
		e.Use(metrics)
	}
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		HandleError: true, LogStatus: true, LogMethod: true, LogRoutePath: true, LogLatency: true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			logger.FromContext(c.Request().Context()).InfoContext(c.Request().Context(), "http request", "method", v.Method, "route", v.RoutePath, "status", v.Status, "latency", v.Latency)
			return nil
		},
	}))
	e.Use(middleware.Recover())
	if len(cfg.CORSOrigins) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{AllowOrigins: cfg.CORSOrigins, AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, echo.HeaderXRequestID}, ExposeHeaders: []string{echo.HeaderXRequestID}}))
	}
	e.Use(middleware.BodyLimit(512 << 10))
	if cfg.RequestTimeout > 0 {
		e.Use(middleware.ContextTimeoutWithConfig(middleware.ContextTimeoutConfig{Timeout: cfg.RequestTimeout, Skipper: func(c *echo.Context) bool {
			return cfg.PprofEnabled && (c.Path() == pprofPath || strings.HasPrefix(c.Path(), pprofPath+"/"))
		}}))
	}
}
