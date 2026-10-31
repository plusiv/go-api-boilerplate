// Package apperrors defines application error categories and safe messages.
package apperrors

import "github.com/rotisserie/eris"

// Kind identifies the application-level meaning of an error.
type Kind string

const (
	// NotFound means the requested resource does not exist.
	NotFound Kind = "not_found"
	// Conflict means the request conflicts with existing state.
	Conflict Kind = "conflict"
	// Validation means the input is invalid.
	Validation Kind = "validation"
	// BusinessRule means a business invariant prevents the operation.
	BusinessRule Kind = "business_rule"
	// RateLimit means the caller exceeded an allowed rate.
	RateLimit Kind = "rate_limit"
	// Gone means a resource is no longer available.
	Gone Kind = "gone"
	// Internal means an unexpected failure whose cause is private.
	Internal Kind = "internal"
)

// Error carries a category, a client-safe message, and an optional private cause.
type Error struct {
	Kind    Kind
	Message string
	Cause   error
}

// Error returns the message and, when present, the cause for server-side use.
func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap allows standard error inspection of the underlying cause.
func (e *Error) Unwrap() error { return e.Cause }

// New constructs an error with a client-safe message for an expected failure.
func New(kind Kind, message string) *Error { return &Error{Kind: kind, Message: message} }

// NewNotFound reports a missing resource.
func NewNotFound(resource string) *Error { return New(NotFound, resource+" not found") }

// NewConflict reports a conflict with existing state.
func NewConflict(message string) *Error { return New(Conflict, message) }

// NewValidation reports invalid input.
func NewValidation(message string) *Error { return New(Validation, message) }

// NewBusinessRule reports a business invariant violation.
func NewBusinessRule(message string) *Error { return New(BusinessRule, message) }

// NewRateLimit reports a rate limit violation.
func NewRateLimit(message string) *Error { return New(RateLimit, message) }

// NewGone reports a resource that is no longer available.
func NewGone(message string) *Error { return New(Gone, message) }

// NewInternal records an unexpected cause and captures its stack trace.
func NewInternal(cause error) *Error {
	return &Error{Kind: Internal, Message: "internal server error", Cause: eris.Wrap(cause, "internal failure")}
}
