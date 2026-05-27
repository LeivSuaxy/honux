# 10 — Error Handling, Abstractions & HTTP Propagation

> The goal is a single, coherent error system that travels cleanly from validation → service → repository → HTTP response, giving clients precise, actionable feedback and giving developers full traceability.

---

## 10.1 The Core Principle

Go errors are values. The strategy is:

1. **Define domain error types** that carry semantic meaning (not just a message string).
2. **Wrap errors** with context as they propagate up the call stack.
3. **Translate domain errors to HTTP** in one central place — the handler layer.

```
Validation / Repository / Service
         ↓  return *AppError / wrap with fmt.Errorf
      Service layer
         ↓  return *AppError / wrap
      Handler layer
         ↓  errors.As(*AppError) → JSON response
      HTTP client
```

---

## 10.2 Domain Error Types

```go
// internal/apperror/apperror.go
package apperror

import (
    "errors"
    "fmt"
    "net/http"
)

// ErrorCode is a machine-readable string code.
// Clients can switch on this without parsing messages.
type ErrorCode string

const (
    CodeNotFound         ErrorCode = "NOT_FOUND"
    CodeConflict         ErrorCode = "CONFLICT"
    CodeUnauthorized     ErrorCode = "UNAUTHORIZED"
    CodeForbidden        ErrorCode = "FORBIDDEN"
    CodeValidation       ErrorCode = "VALIDATION_ERROR"
    CodeInternal         ErrorCode = "INTERNAL_ERROR"
    CodeUnprocessable    ErrorCode = "UNPROCESSABLE_ENTITY"
    CodeBadRequest       ErrorCode = "BAD_REQUEST"
)

// AppError is the central error type for the whole application.
type AppError struct {
    Code       ErrorCode         `json:"code"`
    Message    string            `json:"message"`
    Fields     map[string]string `json:"fields,omitempty"` // per-field validation errors
    HTTPStatus int               `json:"-"`                // never serialised; used by handler
    Cause      error             `json:"-"`                // original error for logging
}

func (e *AppError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap lets errors.Is / errors.As traverse the chain.
func (e *AppError) Unwrap() error { return e.Cause }

// ── Constructors ─────────────────────────────────────────────────────────────

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

// ── Helpers ───────────────────────────────────────────────────────────────────

// Is allows errors.Is to match any *AppError with a specific code.
func Is(err error, code ErrorCode) bool {
    var e *AppError
    if errors.As(err, &e) {
        return e.Code == code
    }
    return false
}

// As extracts *AppError from any error in the chain.
func As(err error) (*AppError, bool) {
    var e *AppError
    return e, errors.As(err, &e)
}
```

---

## 10.3 Validation Errors on DTOs

```go
// internal/dto/validation.go
package dto

import "myapp/internal/apperror"

// FieldErrors accumulates per-field errors during DTO validation.
type FieldErrors map[string]string

func (fe FieldErrors) Add(field, message string) {
    fe[field] = message
}

func (fe FieldErrors) HasErrors() bool {
    return len(fe) > 0
}

// ToAppError converts field errors to an *AppError ready to return from any layer.
func (fe FieldErrors) ToAppError() error {
    if !fe.HasErrors() {
        return nil
    }
    return apperror.ValidationError(fe)
}
```

```go
// internal/dto/user_dto.go
package dto

import (
    "net/mail"
    "strings"
)

// CreateUserRequest is the inbound DTO for POST /users.
type CreateUserRequest struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
    IsAdmin  bool   `json:"is_admin"`
}

// Validate returns nil or a *apperror.AppError with per-field details.
func (r *CreateUserRequest) Validate() error {
    fe := make(FieldErrors)

    if strings.TrimSpace(r.Username) == "" {
        fe.Add("username", "username is required")
    } else if len(r.Username) < 3 || len(r.Username) > 50 {
        fe.Add("username", "username must be between 3 and 50 characters")
    }

    if strings.TrimSpace(r.Email) == "" {
        fe.Add("email", "email is required")
    } else if _, err := mail.ParseAddress(r.Email); err != nil {
        fe.Add("email", "email is not valid")
    }

    if len(r.Password) < 8 {
        fe.Add("password", "password must be at least 8 characters")
    }

    return fe.ToAppError() // nil if no errors
}

// UpdateUserRequest is the inbound DTO for PUT /users/{id}.
type UpdateUserRequest struct {
    Username *string `json:"username"` // pointer = optional field
    Email    *string `json:"email"`
}

func (r *UpdateUserRequest) Validate() error {
    fe := make(FieldErrors)

    if r.Username != nil {
        if len(*r.Username) < 3 || len(*r.Username) > 50 {
            fe.Add("username", "username must be between 3 and 50 characters")
        }
    }

    if r.Email != nil {
        if _, err := mail.ParseAddress(*r.Email); err != nil {
            fe.Add("email", "email is not valid")
        }
    }

    return fe.ToAppError()
}
```

---

## 10.4 Propagating Errors Through Layers

### Repository layer — wrap with context

```go
// internal/repository/user_repository.go
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
    // ...
    err := r.db.QueryRowContext(ctx, query, id).Scan(...)

    if errors.Is(err, sql.ErrNoRows) {
        return nil, apperror.NotFound("user") // semantic error, no cause needed
    }
    if err != nil {
        // Internal DB error — wrap with fmt.Errorf to add context for logs
        return nil, fmt.Errorf("UserRepository.FindByID(id=%s): %w",
            id, apperror.Internal(err))
    }
    return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
    _, err := r.db.ExecContext(ctx, query, u.Username, u.PasswordHash, u.Email, u.IsAdmin)
    if err != nil {
        // Detect unique constraint violation
        if isUniqueViolation(err) {
            return apperror.Conflict("a user with this email already exists", err)
        }
        return fmt.Errorf("UserRepository.Create: %w", apperror.Internal(err))
    }
    return nil
}

// isUniqueViolation detects PostgreSQL error code 23505.
func isUniqueViolation(err error) bool {
    var pgErr *pgconn.PgError
    return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
```

### Service layer — validate, orchestrate, re-wrap

```go
// internal/service/user_service.go
func (s *UserService) Create(ctx context.Context, req *dto.CreateUserRequest) (*model.User, error) {
    // 1. Validate DTO — returns *apperror.AppError if invalid
    if err := req.Validate(); err != nil {
        return nil, err // already a well-formed AppError, just pass it up
    }

    // 2. Business rule check
    existing, err := s.repo.FindByEmail(ctx, req.Email)
    if err != nil {
        return nil, fmt.Errorf("UserService.Create check email: %w", err)
    }
    if existing != nil {
        return nil, apperror.Conflict("email already registered")
    }

    // 3. Hash password
    hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        return nil, fmt.Errorf("UserService.Create hash: %w", apperror.Internal(err))
    }

    user := &model.User{
        Username:     req.Username,
        Email:        req.Email,
        PasswordHash: string(hash),
        IsAdmin:      req.IsAdmin,
    }

    // 4. Persist — repository may return NotFound, Conflict, Internal
    if err := s.repo.Create(ctx, user); err != nil {
        return nil, fmt.Errorf("UserService.Create persist: %w", err)
    }

    return user, nil
}
```

---

## 10.5 HTTP Translation — One Central Place

The handler layer is the **only** place that knows about HTTP. It translates `*AppError` to JSON responses.

```go
// internal/handler/errors.go
package handler

import (
    "errors"
    "log/slog"
    "net/http"

    "myapp/internal/apperror"
)

// HTTPErrorResponse is the JSON shape sent to clients.
type HTTPErrorResponse struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Fields  map[string]string `json:"fields,omitempty"`
}

// respondError is the single function that converts any error to an HTTP response.
func (h *Handler) respondError(w http.ResponseWriter, r *http.Request, err error) {
    var appErr *apperror.AppError

    if errors.As(err, &appErr) {
        // Log internal errors with full cause for observability
        if appErr.HTTPStatus == http.StatusInternalServerError {
            h.logger.Error("internal error",
                "path", r.URL.Path,
                "method", r.Method,
                "error", appErr.Cause,
            )
        }

        h.writeJSON(w, appErr.HTTPStatus, HTTPErrorResponse{
            Code:    string(appErr.Code),
            Message: appErr.Message,
            Fields:  appErr.Fields,
        })
        return
    }

    // Unknown error — never expose internal details to the client
    h.logger.Error("unexpected error",
        "path", r.URL.Path,
        "error", err,
    )
    h.writeJSON(w, http.StatusInternalServerError, HTTPErrorResponse{
        Code:    string(apperror.CodeInternal),
        Message: "an unexpected error occurred",
    })
}
```

```go
// internal/handler/user.go
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req dto.CreateUserRequest
    if err := decodeJSON(w, r, &req); err != nil {
        h.respondError(w, r, apperror.BadRequest(err.Error()))
        return
    }

    user, err := h.users.Create(r.Context(), &req)
    if err != nil {
        h.respondError(w, r, err) // ← single call handles all error types
        return
    }

    h.writeJSON(w, http.StatusCreated, user)
}
```

---

## 10.6 HTTP Response Examples

**Validation error (422):**
```json
{
  "code": "VALIDATION_ERROR",
  "message": "validation failed",
  "fields": {
    "email": "email is not valid",
    "password": "password must be at least 8 characters"
  }
}
```

**Not found (404):**
```json
{
  "code": "NOT_FOUND",
  "message": "user not found"
}
```

**Conflict (409):**
```json
{
  "code": "CONFLICT",
  "message": "email already registered"
}
```

**Internal error (500) — client sees nothing sensitive:**
```json
{
  "code": "INTERNAL_ERROR",
  "message": "an unexpected error occurred"
}
```

---

## 10.7 PostgreSQL Error Code Helper

```go
// internal/repository/pg_errors.go
package repository

import (
    "errors"
    "github.com/jackc/pgx/v5/pgconn"
)

const (
    pgUniqueViolation     = "23505"
    pgForeignKeyViolation = "23503"
    pgNotNullViolation    = "23502"
    pgCheckViolation      = "23514"
)

func pgErrCode(err error) string {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return pgErr.Code
    }
    return ""
}

func isUniqueViolation(err error) bool     { return pgErrCode(err) == pgUniqueViolation }
func isForeignKeyViolation(err error) bool { return pgErrCode(err) == pgForeignKeyViolation }
func isNotNullViolation(err error) bool    { return pgErrCode(err) == pgNotNullViolation }
```

---

## 10.8 Sentinel Errors for Internal Checks

For cases where the service or another repository needs to test a specific condition:

```go
// internal/apperror/sentinel.go
package apperror

import "errors"

// Sentinel values — use errors.Is() to test for these
var (
    ErrNotFound     = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
    ErrConflict     = errors.New("conflict")
)
```

```go
// Usage
if apperror.Is(err, apperror.CodeNotFound) {
    // handle specifically
}

// Or with errors.As for the full struct
if appErr, ok := apperror.As(err); ok {
    log.Println(appErr.Fields) // access per-field details
}
```

---

## 10.9 Full Error Flow Diagram

```
POST /users  (invalid body)
     │
     ▼
Handler.CreateUser
     │  decodeJSON → bad JSON → apperror.BadRequest("...")
     │  dto.Validate() → apperror.ValidationError(fields)
     ▼
UserService.Create
     │  repo.FindByEmail → apperror.NotFound / Internal
     │  business rule → apperror.Conflict(...)
     ▼
UserRepository.Create
     │  sql unique violation → apperror.Conflict(...)
     │  other DB error → fmt.Errorf("repo: %w", apperror.Internal(err))
     ▼
Handler.respondError(w, r, err)
     │  errors.As(*AppError) → write JSON with HTTPStatus
     │  unknown error → 500, log full cause
     ▼
Client receives structured JSON
```

---

## 10.10 Summary Table

| Layer | Responsibility | Error Action |
|---|---|---|
| DTO / Validation | Check shape and rules of input | Return `apperror.ValidationError(fields)` |
| Repository | DB access only | Wrap DB errors → `apperror.NotFound`, `apperror.Conflict`, `apperror.Internal` |
| Service | Orchestrate + business rules | Pass through or wrap with `fmt.Errorf("context: %w", err)` |
| Handler | HTTP translation | `errors.As(*AppError)` → JSON; unknown → 500 + log |

---

*Next: [11 — Repository Relations & Optimal SQL →](./11-repository-relations-and-sql.md)*
