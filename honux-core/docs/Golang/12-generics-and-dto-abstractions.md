# 12 — Generics & DTO Abstractions in Go

> Go 1.18 introduced generics (type parameters). This guide shows how to use them to eliminate repetition in DTOs, repositories, and response envelopes — without sacrificing readability or Go's explicit style.

---

## 12.1 Generics Syntax Refresher

```go
// T is a type parameter. [T any] means T can be any type.
func Map[T, U any](slice []T, fn func(T) U) []U {
    out := make([]U, len(slice))
    for i, v := range slice {
        out[i] = fn(v)
    }
    return out
}

// Constrained type parameter — T must be int, int64, or float64
type Number interface {
    int | int64 | float64
}

func Sum[T Number](nums []T) T {
    var total T
    for _, n := range nums {
        total += n
    }
    return total
}
```

---

## 12.2 The DTO Contract — Interfaces as Constraints

Define the two behaviours every DTO must have as an interface:

```go
// internal/dto/contract.go
package dto

// Request is the generic constraint for all inbound DTOs.
// T is the schema (domain struct) this DTO converts into.
type Request[T any] interface {
    Validate() error   // returns *apperror.AppError or nil
    ToSchema() T       // converts DTO → internal schema
}
```

This interface can be used as a type constraint anywhere you want to handle a DTO generically.

---

## 12.3 Concrete DTO Implementation

```go
// internal/dto/user_dto.go
package dto

import (
    "net/mail"
    "strings"

    "myapp/internal/apperror"
    "myapp/internal/schema"
)

// ── Create ────────────────────────────────────────────────────────────────────

type CreateUserRequest struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
    IsAdmin  bool   `json:"is_admin"`
}

func (r *CreateUserRequest) Validate() error {
    fe := make(FieldErrors)

    if strings.TrimSpace(r.Username) == "" {
        fe.Add("username", "username is required")
    } else if l := len(r.Username); l < 3 || l > 50 {
        fe.Add("username", "must be between 3 and 50 characters")
    }

    if strings.TrimSpace(r.Email) == "" {
        fe.Add("email", "email is required")
    } else if _, err := mail.ParseAddress(r.Email); err != nil {
        fe.Add("email", "email is not valid")
    }

    if len(r.Password) < 8 {
        fe.Add("password", "password must be at least 8 characters")
    }

    return fe.ToAppError()
}

// ToSchema converts the DTO to the internal schema consumed by services.
// Password hashing happens in the service, not here.
func (r *CreateUserRequest) ToSchema() schema.CreateUser {
    return schema.CreateUser{
        Username: strings.TrimSpace(r.Username),
        Email:    strings.ToLower(strings.TrimSpace(r.Email)),
        Password: r.Password,
        IsAdmin:  r.IsAdmin,
    }
}

// ── Update ────────────────────────────────────────────────────────────────────

type UpdateUserRequest struct {
    Username *string `json:"username"`
    Email    *string `json:"email"`
}

func (r *UpdateUserRequest) Validate() error {
    fe := make(FieldErrors)

    if r.Username != nil {
        if l := len(*r.Username); l < 3 || l > 50 {
            fe.Add("username", "must be between 3 and 50 characters")
        }
    }
    if r.Email != nil {
        if _, err := mail.ParseAddress(*r.Email); err != nil {
            fe.Add("email", "email is not valid")
        }
    }

    return fe.ToAppError()
}

func (r *UpdateUserRequest) ToSchema() schema.UpdateUser {
    return schema.UpdateUser{
        Username: r.Username,
        Email:    r.Email,
    }
}
```

```go
// internal/schema/user_schema.go
package schema

// Schemas are the internal data shapes that services and repositories consume.
// They are decoupled from both HTTP DTOs and database models.

type CreateUser struct {
    Username string
    Email    string
    Password string
    IsAdmin  bool
}

type UpdateUser struct {
    Username *string
    Email    *string
}
```

---

## 12.4 Generic Handler Helper — Decode, Validate, and Extract in One Call

This eliminates the decode + validate boilerplate from every handler.

```go
// internal/handler/bind.go
package handler

import (
    "net/http"

    "myapp/internal/apperror"
    "myapp/internal/dto"
)

// Bind decodes the JSON body into req, validates it, and returns the schema.
// On any failure it writes the appropriate error response and returns false.
// T = schema type (e.g. schema.CreateUser)
// R = DTO type that satisfies dto.Request[T]
func Bind[T any, R dto.Request[T]](
    h *Handler,
    w http.ResponseWriter,
    r *http.Request,
    req R,
) (T, bool) {
    var zero T

    if err := decodeJSON(w, r, req); err != nil {
        h.respondError(w, r, apperror.BadRequest(err.Error()))
        return zero, false
    }

    if err := req.Validate(); err != nil {
        h.respondError(w, r, err)
        return zero, false
    }

    return req.ToSchema(), true
}
```

```go
// internal/handler/user.go — handler becomes one-liners
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    s, ok := Bind[schema.CreateUser](h, w, r, &dto.CreateUserRequest{})
    if !ok {
        return
    }

    user, err := h.users.Create(r.Context(), s)
    if err != nil {
        h.respondError(w, r, err)
        return
    }

    h.writeJSON(w, http.StatusCreated, user)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
    id, err := parseUUID(r, "id")
    if err != nil {
        h.respondError(w, r, apperror.BadRequest("invalid id"))
        return
    }

    s, ok := Bind[schema.UpdateUser](h, w, r, &dto.UpdateUserRequest{})
    if !ok {
        return
    }

    user, err := h.users.Update(r.Context(), id, s)
    if err != nil {
        h.respondError(w, r, err)
        return
    }

    h.writeJSON(w, http.StatusOK, user)
}
```

---

## 12.5 Generic Repository Interface

Define a common CRUD interface any repository can satisfy:

```go
// internal/repository/repository.go
package repository

import (
    "context"

    "github.com/google/uuid"
)

// CRUD is a generic base contract for repositories.
// M = model type (e.g. *model.User)
// S = create schema type (e.g. schema.CreateUser)
type CRUD[M any, S any] interface {
    FindByID(ctx context.Context, id uuid.UUID) (M, error)
    FindAll(ctx context.Context, page Page) (*PagedResult[M], error)
    Create(ctx context.Context, s S) (M, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

Your `UserRepository` can satisfy this interface implicitly by implementing all four methods. You don't need to declare it explicitly.

---

## 12.6 Generic Slice Utilities

Reusable helpers that work on any slice — no more duplicated `for` loops.

```go
// internal/util/slice.go
package util

// Map transforms []T to []U using fn.
func Map[T, U any](s []T, fn func(T) U) []U {
    out := make([]U, len(s))
    for i, v := range s {
        out[i] = fn(v)
    }
    return out
}

// Filter returns elements of s for which fn returns true.
func Filter[T any](s []T, fn func(T) bool) []T {
    var out []T
    for _, v := range s {
        if fn(v) {
            out = append(out, v)
        }
    }
    return out
}

// Find returns the first element matching fn, or the zero value + false.
func Find[T any](s []T, fn func(T) bool) (T, bool) {
    for _, v := range s {
        if fn(v) {
            return v, true
        }
    }
    var zero T
    return zero, false
}

// GroupBy groups elements by the key returned by fn.
func GroupBy[T any, K comparable](s []T, fn func(T) K) map[K][]T {
    m := make(map[K][]T)
    for _, v := range s {
        k := fn(v)
        m[k] = append(m[k], v)
    }
    return m
}

// ToMap converts a slice to a map keyed by fn.
func ToMap[T any, K comparable](s []T, fn func(T) K) map[K]T {
    m := make(map[K]T, len(s))
    for _, v := range s {
        m[fn(v)] = v
    }
    return m
}

// Contains reports whether s contains v.
func Contains[T comparable](s []T, v T) bool {
    for _, item := range s {
        if item == v {
            return true
        }
    }
    return false
}

// Unique returns s with duplicates removed, preserving order.
func Unique[T comparable](s []T) []T {
    seen := make(map[T]struct{}, len(s))
    out  := make([]T, 0, len(s))
    for _, v := range s {
        if _, ok := seen[v]; !ok {
            seen[v] = struct{}{}
            out = append(out, v)
        }
    }
    return out
}
```

```go
// Usage examples
users := []*model.User{ ... }

// Extract IDs for a batch query
ids := util.Map(users, func(u *model.User) uuid.UUID { return u.ID })

// Filter only active users
active := util.Filter(users, func(u *model.User) bool { return u.Active })

// Group users by admin status
byAdmin := util.GroupBy(users, func(u *model.User) bool { return u.IsAdmin })

// Fast lookup map
byID := util.ToMap(users, func(u *model.User) uuid.UUID { return u.ID })
user := byID[someID]
```

---

## 12.7 Generic API Response Envelope

```go
// internal/handler/response.go
package handler

// Envelope wraps any response body in a consistent JSON shape.
type Envelope[T any] struct {
    Data T      `json:"data"`
    Meta *Meta  `json:"meta,omitempty"`
}

type Meta struct {
    TotalCount int `json:"total_count,omitempty"`
    Limit      int `json:"limit,omitempty"`
    Offset     int `json:"offset,omitempty"`
}

// writeData writes a typed Envelope.
func (h *Handler) writeData[T any](w http.ResponseWriter, status int, data T, meta ...*Meta) {
    env := Envelope[T]{Data: data}
    if len(meta) > 0 {
        env.Meta = meta[0]
    }
    h.writeJSON(w, status, env)
}
```

```go
// Single user response
h.writeData(w, http.StatusOK, user)

// Paged list response
h.writeData(w, http.StatusOK, users, &Meta{
    TotalCount: result.TotalCount,
    Limit:      result.Limit,
    Offset:     result.Offset,
})
```

**Response shape:**
```json
{
  "data": { "id": "...", "username": "alice" },
  "meta": { "total_count": 42, "limit": 20, "offset": 0 }
}
```

---

## 12.8 Generic Optional (Result Type)

A lightweight result type for operations that might return nothing:

```go
// internal/util/optional.go
package util

type Optional[T any] struct {
    value *T
}

func Some[T any](v T) Optional[T]  { return Optional[T]{value: &v} }
func None[T any]() Optional[T]     { return Optional[T]{} }

func (o Optional[T]) IsPresent() bool   { return o.value != nil }
func (o Optional[T]) Get() (T, bool) {
    if o.value == nil {
        var zero T
        return zero, false
    }
    return *o.value, true
}

func (o Optional[T]) OrElse(def T) T {
    if o.value == nil {
        return def
    }
    return *o.value
}

// Usage
result := repo.FindByEmail(ctx, email) // returns Optional[*model.User]
if user, ok := result.Get(); ok {
    // user exists
}
```

---

## 12.9 Generic Cache

```go
// internal/util/cache.go
package util

import (
    "sync"
    "time"
)

type entry[V any] struct {
    value     V
    expiresAt time.Time
}

type TTLCache[K comparable, V any] struct {
    mu   sync.RWMutex
    data map[K]entry[V]
    ttl  time.Duration
}

func NewTTLCache[K comparable, V any](ttl time.Duration) *TTLCache[K, V] {
    return &TTLCache[K, V]{data: make(map[K]entry[V]), ttl: ttl}
}

func (c *TTLCache[K, V]) Get(key K) (V, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    e, ok := c.data[key]
    if !ok || time.Now().After(e.expiresAt) {
        var zero V
        return zero, false
    }
    return e.value, true
}

func (c *TTLCache[K, V]) Set(key K, value V) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[key] = entry[V]{value: value, expiresAt: time.Now().Add(c.ttl)}
}

func (c *TTLCache[K, V]) Delete(key K) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.data, key)
}
```

```go
// Usage — type-safe cache, no interface{} casting
userCache := util.NewTTLCache[uuid.UUID, *model.User](5 * time.Minute)
userCache.Set(user.ID, user)
if u, ok := userCache.Get(id); ok {
    return u, nil
}
```

---

## 12.10 Constraints Reference

```go
// Common constraint interfaces from golang.org/x/exp/constraints
// (or define your own)

type Integer interface {
    int | int8 | int16 | int32 | int64 |
    uint | uint8 | uint16 | uint32 | uint64
}

type Float interface {
    float32 | float64
}

type Ordered interface {
    Integer | Float | ~string
}

// ~T means "any type whose underlying type is T"
type Stringish interface {
    ~string
}

// Usage with ~
type UserID string
// UserID satisfies Stringish because its underlying type is string
```

---

## 12.11 What Generics Are Not Good For in Go

| Avoid | Use Instead |
|---|---|
| Generic HTTP handlers for every entity | Explicit handlers — clarity > brevity |
| Generic service layer with one `Save(T)` | Explicit service methods per entity |
| Replacing interfaces entirely | Interfaces for behaviour; generics for data containers |
| Complex type algebra | Keep it simple — if the constraint is hard to read, it's probably wrong |

The sweet spots for generics in Go: **collections/utilities, response envelopes, caches, pagination wrappers, and result types** — anything data-shaped. Business logic stays explicit.

---

*Next: [13 — Testing in Go →](./13-testing.md)*
