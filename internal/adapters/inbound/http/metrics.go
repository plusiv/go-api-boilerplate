package http

import (
	"fmt"
	"net/http"
	"strings"

	echoprometheus "github.com/labstack/echo-prometheus"
	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func registerMetrics(e *echo.Echo, token string) (echo.MiddlewareFunc, error) {
	registry := prometheus.NewRegistry()
	for _, collector := range []prometheus.Collector{collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}), collectors.NewBuildInfoCollector()} {
		if err := registry.Register(collector); err != nil {
			return nil, fmt.Errorf("register runtime metrics: %w", err)
		}
	}
	config := echoprometheus.MiddlewareConfig{
		Registerer: registry, DoNotUseRequestPathFor404: true,
		Skipper: func(c *echo.Context) bool {
			path := c.Request().URL.Path
			return path == "/metrics" || path == pprofPath || strings.HasPrefix(path, pprofPath+"/")
		},
		// Host and method are supplied by clients, so bound their values to avoid unbounded labels.
		LabelFuncs: map[string]echoprometheus.LabelValueFunc{
			"host": func(_ *echo.Context, _ error) string { return "" },
			"method": func(c *echo.Context, _ error) string {
				switch c.Request().Method {
				case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
					return c.Request().Method
				default:
					return "OTHER"
				}
			},
		},
	}
	middleware, err := config.ToMiddleware()
	if err != nil {
		return nil, fmt.Errorf("register HTTP metrics: %w", err)
	}
	e.GET("/metrics", echo.WrapHandler(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})), bearerAuth(token))
	return middleware, nil
}
