package http

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/labstack/echo/v5"
)

const pprofPath = "/metrics/pprof"

func registerPprof(e *echo.Echo, token string) {
	group := e.Group(pprofPath, bearerAuth(token))
	group.GET("", func(c *echo.Context) error { return c.Redirect(http.StatusPermanentRedirect, pprofPath+"/") })
	group.GET("/", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	group.GET("/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	group.GET("/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	group.GET("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	group.POST("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	group.GET("/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))
	group.GET("/:name", func(c *echo.Context) error {
		pprof.Handler(c.Param("name")).ServeHTTP(c.Response(), c.Request())
		return nil
	})
}

func bearerAuth(token string) echo.MiddlewareFunc {
	expected := sha256.Sum256([]byte(token))
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			value, ok := strings.CutPrefix(c.Request().Header.Get(echo.HeaderAuthorization), "Bearer ")
			actual := sha256.Sum256([]byte(value))
			if !ok || subtle.ConstantTimeCompare(actual[:], expected[:]) != 1 {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}
			return next(c)
		}
	}
}
