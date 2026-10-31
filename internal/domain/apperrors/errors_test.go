package apperrors

import (
	"errors"
	"strings"
	"testing"

	"github.com/rotisserie/eris"
)

func TestErrors(t *testing.T) {
	for _, tt := range []struct {
		name    string
		err     *Error
		kind    Kind
		message string
	}{
		{"not found", NewNotFound("item"), NotFound, "item not found"},
		{"conflict", NewConflict("exists"), Conflict, "exists"},
		{"validation", NewValidation("invalid"), Validation, "invalid"},
		{"business", NewBusinessRule("not allowed"), BusinessRule, "not allowed"},
		{"rate limit", NewRateLimit("too many"), RateLimit, "too many"},
		{"gone", NewGone("removed"), Gone, "removed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Kind != tt.kind || tt.err.Error() != tt.message {
				t.Fatalf("got %v", tt.err)
			}
			var found *Error
			if !errors.As(eris.Wrap(tt.err, "operation"), &found) || found != tt.err {
				t.Fatal("error cannot be unwrapped")
			}
		})
	}
	cause := errors.New("private detail")
	internal := NewInternal(cause)
	if internal.Kind != Internal || internal.Message != "internal server error" || !errors.Is(internal, cause) || !strings.Contains(eris.ToString(internal, true), "private detail") {
		t.Fatalf("internal error %v", internal)
	}
}
