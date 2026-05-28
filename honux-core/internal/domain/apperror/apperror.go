package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

type ErrorCode string

const (
	CodeNotFound      ErrorCode = "NOT_FOUND"
	CodeConflict      ErrorCode = "CONFLICT"
	CodeUnauthorized  ErrorCode = "UNAUTHORIZED"
	CodeForbidden     ErrorCode = "FORBIDDEN"
	CodeValidation    ErrorCode = "VALIDATION_ERROR"
	CodeInternal      ErrorCode = "INTERNAL_ERROR"
	CodeUnprocessable ErrorCode = "UNPROCESSABLE_ENTITY"
	CodeBadRequest    ErrorCode = "BAD_REQUEST"
)

type AppError struct {
	Code       ErrorCode         `json:"code"`
	Message    string            `json:"message"`
	Fields     map[string]string `json:"fields,omitempty"`
	HTTPStatus int               `json:"-"`
	Cause      error             `json:"-"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Cause }

// --Constructors--

func NotFound(resource string, cause ...error) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
		Cause:      firstOrNil(cause),
	}
}

func Conflict(message string, cause ...error) *AppError {
	return &AppError{
		Code:       CodeConflict,
		Message:    message,
		HTTPStatus: http.StatusConflict,
		Cause:      firstOrNil(cause),
	}
}

func Unauthorized(message string) *AppError {
	return &AppError{
		Code:       CodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

func Forbidden(message string) *AppError {
	return &AppError{
		Code:       CodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

func Internal(cause error) *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    "an unexpected error occurred",
		HTTPStatus: http.StatusInternalServerError,
		Cause:      cause,
	}
}

func BadRequest(message string, cause ...error) *AppError {
	return &AppError{
		Code:       CodeBadRequest,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Cause:      firstOrNil(cause),
	}
}

// ValidationError builds an AppError with per-field error details.
func ValidationError(fields map[string]string) *AppError {
	return &AppError{
		Code:       CodeValidation,
		Message:    "validation failed",
		Fields:     fields,
		HTTPStatus: http.StatusUnprocessableEntity,
	}
}

func firstOrNil(errs []error) error {
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func Is(err error, code ErrorCode) bool {
	if e, ok := errors.AsType[*AppError](err); ok {
		return e.Code == code
	}
	return false
}

func As(err error) (*AppError, bool) {
	var e *AppError
	return e, errors.As(err, &e)
}
