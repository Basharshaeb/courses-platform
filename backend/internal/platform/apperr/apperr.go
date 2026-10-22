// Package apperr defines errors that map to RFC 9457 problem responses.
package apperr

import (
	"errors"
	"net/http"
)

// Error is an expected, client-facing error. Code is stable and machine readable.
type Error struct {
	Status int
	Code   string
	Detail string
}

func (e *Error) Error() string { return e.Code + ": " + e.Detail }

func New(status int, code, detail string) *Error {
	return &Error{Status: status, Code: code, Detail: detail}
}

func BadRequest(code, detail string) *Error { return New(http.StatusBadRequest, code, detail) }
func Unauthorized(detail string) *Error     { return New(http.StatusUnauthorized, "unauthorized", detail) }
func Forbidden(detail string) *Error        { return New(http.StatusForbidden, "forbidden", detail) }
func NotFound(code, detail string) *Error   { return New(http.StatusNotFound, code, detail) }
func Conflict(code, detail string) *Error   { return New(http.StatusConflict, code, detail) }

// As extracts an *Error from err.
func As(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
