package http

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/plusiv/go-api-boilerplate/internal/domain/apperrors"
	"github.com/plusiv/go-api-boilerplate/internal/infrastructure/logger"
	"github.com/rotisserie/eris"
)

func handleError(c *echo.Context, err error) {
	if resp, _ := echo.UnwrapResponse(c.Response()); resp != nil && resp.Committed {
		return
	}
	status, message := http.StatusInternalServerError, "internal server error"
	var fields []string
	var appErr *apperrors.Error
	var validation *validationError
	switch {
	case errors.As(err, &appErr):
		switch appErr.Kind {
		case apperrors.NotFound:
			status = http.StatusNotFound
		case apperrors.Conflict:
			status = http.StatusConflict
		case apperrors.Validation, apperrors.BusinessRule:
			status = http.StatusUnprocessableEntity
		case apperrors.RateLimit:
			status = http.StatusTooManyRequests
		case apperrors.Gone:
			status = http.StatusGone
		default:
			status = http.StatusInternalServerError
		}
		if status != http.StatusInternalServerError {
			message = appErr.Message
		}
		if appErr.Kind == apperrors.Validation && errors.As(err, &validation) {
			fields = validation.fields
		}
	case errors.Is(err, echo.ErrStatusRequestEntityTooLarge):
		status, message = http.StatusRequestEntityTooLarge, http.StatusText(http.StatusRequestEntityTooLarge)
	default:
		if code := echo.StatusCode(err); code >= 400 && code <= 599 {
			status, message = code, http.StatusText(code)
		}
	}
	log := logger.FromContext(c.Request().Context())
	if status >= 500 {
		log.ErrorContext(c.Request().Context(), "request failed", "error", eris.ToString(err, true))
	}
	if writeErr := c.JSON(status, errorResponse{Error: message, Fields: fields}); writeErr != nil {
		log.ErrorContext(c.Request().Context(), "write error response", "error", writeErr)
	}
}
