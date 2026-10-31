package http

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/plusiv/go-api-boilerplate/internal/domain/apperrors"
)

type validateBinder struct{ validator *validator.Validate }

type validationError struct {
	cause  *apperrors.Error
	fields []string
}

func (e *validationError) Error() string { return e.cause.Error() }
func (e *validationError) Unwrap() error { return e.cause }

func newValidateBinder() (*validateBinder, error) {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		for _, tag := range []string{"json", "query", "param", "form"} {
			name := strings.SplitN(field.Tag.Get(tag), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
		}
		return field.Name
	})
	if err := v.RegisterValidation("notblank", func(field validator.FieldLevel) bool { return strings.TrimSpace(field.Field().String()) != "" }); err != nil {
		return nil, fmt.Errorf("register validation: %w", err)
	}
	return &validateBinder{validator: v}, nil
}

func (b *validateBinder) Bind(c *echo.Context, target any) error {
	if err := (&echo.DefaultBinder{}).Bind(c, target); err != nil {
		return err
	}
	if err := b.validator.StructCtx(c.Request().Context(), target); err != nil {
		var fields validator.ValidationErrors
		if !errors.As(err, &fields) {
			return fmt.Errorf("validate request: %w", err)
		}
		result := &validationError{cause: apperrors.NewValidation("invalid input"), fields: make([]string, 0, len(fields))}
		for _, field := range fields {
			result.fields = append(result.fields, field.Field()+": "+field.Tag())
		}
		return result
	}
	return nil
}
