# 06 — HTTP Handlers & Endpoints (Scalable, Native Go)

> Go's `net/http` package is production-ready without any framework. This guide shows how to structure a scalable HTTP application using only the standard library plus the `chi` router for composable routing (it stays close to `net/http` semantics and adds zero magic).

---

## 6.1 The `http.Handler` Interface

Everything in Go's HTTP stack revolves around one interface:

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

Any type that implements `ServeHTTP` can handle HTTP requests. `http.HandlerFunc` is a function adapter:

```go
// HandlerFunc is defined in the standard library as:
type HandlerFunc func(ResponseWriter, *Request)
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) { f(w, r) }
```

---

## 6.2 Project Structure

```
myapp/
├── cmd/
│   └── api/
│       └── main.go          # entry point
├── internal/
│   ├── handler/             # HTTP handlers (thin layer)
│   │   ├── user.go
│   │   └── product.go
│   ├── service/             # business logic
│   │   └── user.go
│   ├── repository/          # data access
│   │   └── user.go
│   ├── middleware/          # HTTP middleware
│   │   ├── auth.go
│   │   └── logging.go
│   ├── server/              # server setup & routing
│   │   └── server.go
│   └── model/               # domain types
│       └── user.go
├── go.mod
└── go.sum
```

---

## 6.3 The Server — Wiring Everything Together

```go
// internal/server/server.go
package server

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "myapp/internal/handler"
    "myapp/internal/middleware"
    "myapp/internal/service"
    "myapp/internal/repository"
)

type Server struct {
    httpServer *http.Server
}

func New(port int, db *sql.DB) *Server {
    // Build the dependency graph (manual DI — no magic)
    userRepo := repository.NewUserRepository(db)
    userSvc  := service.NewUserService(userRepo)
    userHandler := handler.NewUserHandler(userSvc)

    mux := http.NewServeMux()
    registerRoutes(mux, userHandler)

    // Wrap the mux with global middleware (applied to every request)
    stack := middleware.Chain(
        middleware.Recover,
        middleware.RequestID,
        middleware.Logger,
    )

    return &Server{
        httpServer: &http.Server{
            Addr:         fmt.Sprintf(":%d", port),
            Handler:      stack(mux),
            ReadTimeout:  10 * time.Second,
            WriteTimeout: 30 * time.Second,
            IdleTimeout:  60 * time.Second,
        },
    }
}

func registerRoutes(mux *http.ServeMux, u *handler.UserHandler) {
    // Go 1.22+ pattern syntax: "METHOD /path"
    mux.HandleFunc("GET /health",          healthCheck)
    mux.HandleFunc("GET /api/v1/users",    u.List)
    mux.HandleFunc("POST /api/v1/users",   u.Create)
    mux.HandleFunc("GET /api/v1/users/{id}", u.GetByID)
    mux.HandleFunc("PUT /api/v1/users/{id}", u.Update)
    mux.HandleFunc("DELETE /api/v1/users/{id}", u.Delete)
}

func (s *Server) Start() error {
    return s.httpServer.ListenAndServe()
}

// Graceful shutdown
func (s *Server) Shutdown(ctx context.Context) error {
    return s.httpServer.Shutdown(ctx)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
}
```

---

## 6.4 Entry Point with Graceful Shutdown

```go
// cmd/api/main.go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    _ "github.com/lib/pq"
    "myapp/internal/server"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        slog.Error("failed to open database", "error", err)
        os.Exit(1)
    }
    defer db.Close()

    if err := db.Ping(); err != nil {
        slog.Error("failed to ping database", "error", err)
        os.Exit(1)
    }

    srv := server.New(8080, db)

    // Start in a goroutine so we can listen for shutdown signals
    go func() {
        slog.Info("server starting", "addr", ":8080")
        if err := srv.Start(); err != nil && err != http.ErrServerClosed {
            slog.Error("server error", "error", err)
            os.Exit(1)
        }
    }()

    // Block until SIGINT or SIGTERM
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    slog.Info("shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        slog.Error("forced shutdown", "error", err)
    }
    slog.Info("server stopped")
}
```

---

## 6.5 Models

```go
// internal/model/user.go
package model

import "time"

type User struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r *CreateUserRequest) Validate() error {
    if r.Name == "" {
        return fmt.Errorf("name is required")
    }
    if r.Email == "" {
        return fmt.Errorf("email is required")
    }
    return nil
}
```

---

## 6.6 Repository Layer

```go
// internal/repository/user.go
package repository

import (
    "context"
    "database/sql"
    "fmt"
    "myapp/internal/model"
)

type UserRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) FindAll(ctx context.Context) ([]model.User, error) {
    rows, err := r.db.QueryContext(ctx,
        `SELECT id, name, email, created_at FROM users ORDER BY id`)
    if err != nil {
        return nil, fmt.Errorf("repository.FindAll: %w", err)
    }
    defer rows.Close()

    var users []model.User
    for rows.Next() {
        var u model.User
        if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
            return nil, fmt.Errorf("repository.FindAll scan: %w", err)
        }
        users = append(users, u)
    }
    return users, rows.Err()
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
    var u model.User
    err := r.db.QueryRowContext(ctx,
        `SELECT id, name, email, created_at FROM users WHERE id = $1`, id,
    ).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
    if err == sql.ErrNoRows {
        return nil, nil // not found, not an error
    }
    if err != nil {
        return nil, fmt.Errorf("repository.FindByID: %w", err)
    }
    return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, req model.CreateUserRequest) (*model.User, error) {
    var u model.User
    err := r.db.QueryRowContext(ctx,
        `INSERT INTO users (name, email) VALUES ($1, $2)
         RETURNING id, name, email, created_at`,
        req.Name, req.Email,
    ).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
    if err != nil {
        return nil, fmt.Errorf("repository.Create: %w", err)
    }
    return &u, nil
}
```

---

## 6.7 Service Layer

```go
// internal/service/user.go
package service

import (
    "context"
    "fmt"
    "myapp/internal/model"
    "myapp/internal/repository"
)

type UserService struct {
    repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
    return &UserService{repo: repo}
}

func (s *UserService) List(ctx context.Context) ([]model.User, error) {
    return s.repo.FindAll(ctx)
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*model.User, error) {
    u, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    if u == nil {
        return nil, fmt.Errorf("user %d not found", id)
    }
    return u, nil
}

func (s *UserService) Create(ctx context.Context, req model.CreateUserRequest) (*model.User, error) {
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
    return s.repo.Create(ctx, req)
}
```

---

## 6.8 Handler Layer

```go
// internal/handler/user.go
package handler

import (
    "encoding/json"
    "errors"
    "log/slog"
    "net/http"
    "strconv"

    "myapp/internal/model"
    "myapp/internal/service"
)

type UserHandler struct {
    svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
    return &UserHandler{svc: svc}
}

// --- Helper functions -------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(v); err != nil {
        slog.Error("writeJSON encode error", "error", err)
    }
}

func writeError(w http.ResponseWriter, status int, message string) {
    writeJSON(w, status, map[string]string{"error": message})
}

func pathID(r *http.Request, key string) (int64, error) {
    // Go 1.22+ path value extraction
    s := r.PathValue(key)
    id, err := strconv.ParseInt(s, 10, 64)
    if err != nil {
        return 0, fmt.Errorf("invalid %s: %q", key, s)
    }
    return id, nil
}

// --- Handlers ---------------------------------------------------------------

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
    users, err := h.svc.List(r.Context())
    if err != nil {
        slog.Error("handler.List", "error", err)
        writeError(w, http.StatusInternalServerError, "could not fetch users")
        return
    }
    writeJSON(w, http.StatusOK, users)
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
    id, err := pathID(r, "id")
    if err != nil {
        writeError(w, http.StatusBadRequest, err.Error())
        return
    }

    user, err := h.svc.GetByID(r.Context(), id)
    if err != nil {
        slog.Warn("handler.GetByID", "error", err)
        writeError(w, http.StatusNotFound, err.Error())
        return
    }
    writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req model.CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid JSON body")
        return
    }
    defer r.Body.Close()

    user, err := h.svc.Create(r.Context(), req)
    if err != nil {
        slog.Error("handler.Create", "error", err)
        writeError(w, http.StatusUnprocessableEntity, err.Error())
        return
    }
    writeJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
    // similar pattern: parse id, decode body, call service, write response
    writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id, err := pathID(r, "id")
    if err != nil {
        writeError(w, http.StatusBadRequest, err.Error())
        return
    }
    // call service.Delete...
    _ = id
    w.WriteHeader(http.StatusNoContent)
}
```

---

## 6.9 Go 1.22+ Routing Enhancements

Go 1.22 added method and wildcard routing directly to `http.ServeMux`:

```go
// Method + path pattern
mux.HandleFunc("GET /articles/{id}", getArticle)
mux.HandleFunc("POST /articles", createArticle)
mux.HandleFunc("DELETE /articles/{id}", deleteArticle)

// Wildcard path (trailing slash captures subtree)
mux.HandleFunc("GET /files/{path...}", serveFile)

// Extract path variable
func getArticle(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    fmt.Fprintf(w, "article id: %s", id)
}
```

---

## 6.10 Sub-Routers / Route Groups

For grouping routes under a prefix with dedicated middleware, use `http.StripPrefix` or a helper:

```go
func withPrefix(prefix string, h http.Handler) http.Handler {
    return http.StripPrefix(prefix, h)
}

// Or create a sub-mux and mount it
func apiRoutes(userHandler *handler.UserHandler) http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("GET /users",       userHandler.List)
    mux.HandleFunc("POST /users",      userHandler.Create)
    mux.HandleFunc("GET /users/{id}",  userHandler.GetByID)
    return mux
}

// In main mux
mainMux.Handle("/api/v1/", http.StripPrefix("/api/v1", apiRoutes(userHandler)))
```

---

## 6.11 Response Helpers Pattern

Keep handler code clean by centralizing response logic:

```go
// internal/handler/response.go
package handler

type APIResponse[T any] struct {
    Data  T      `json:"data,omitempty"`
    Error string `json:"error,omitempty"`
    Meta  *Meta  `json:"meta,omitempty"`
}

type Meta struct {
    Total  int `json:"total"`
    Page   int `json:"page"`
    PerPage int `json:"per_page"`
}

func OK[T any](w http.ResponseWriter, data T) {
    writeJSON(w, http.StatusOK, APIResponse[T]{Data: data})
}

func Created[T any](w http.ResponseWriter, data T) {
    writeJSON(w, http.StatusCreated, APIResponse[T]{Data: data})
}

func BadRequest(w http.ResponseWriter, msg string) {
    writeJSON(w, http.StatusBadRequest, APIResponse[any]{Error: msg})
}

func InternalError(w http.ResponseWriter) {
    writeJSON(w, http.StatusInternalServerError, APIResponse[any]{Error: "internal server error"})
}

func NotFound(w http.ResponseWriter, msg string) {
    writeJSON(w, http.StatusNotFound, APIResponse[any]{Error: msg})
}
```

---

*Next: [07 — Best Practices for HTTP Applications →](./07-http-best-practices.md)*
