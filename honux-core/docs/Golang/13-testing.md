# 13 — Testing in Go

> Go ships a complete testing toolkit in the standard library. This guide covers unit tests, integration tests against a real PostgreSQL database (using Docker), HTTP handler tests, mock generation, and a full CRUD test flow — with minimal external dependencies.

---

## 13.1 The Testing Pyramid in Go

```
         ┌──────────┐
         │   E2E    │  ← few, slow, real infra
         ├──────────┤
         │  Integ   │  ← repository tests against real DB (testcontainers)
         ├──────────┤
         │   Unit   │  ← fast, no I/O, mocks for dependencies
         └──────────┘
```

**Rules of thumb:**
- Unit tests: business logic, validation, utilities — no I/O.
- Integration tests: repositories and DB queries — real PostgreSQL in Docker.
- Handler tests: `net/http/httptest` — mock services, no real DB.
- E2E tests: full stack — usually skipped in CI unless critical paths.

---

## 13.2 Go Test Basics

```go
// File must end in _test.go
// Package: same package (white-box) or package_test (black-box)

// go test ./...              run all tests
// go test -run TestUserService ./internal/service/...
// go test -v -race ./...     verbose + race detector
// go test -cover ./...       coverage report
```

---

## 13.3 Table-Driven Tests

The idiomatic Go pattern for exhaustive unit tests:

```go
// internal/dto/user_dto_test.go
package dto_test

import (
    "testing"

    "myapp/internal/apperror"
    "myapp/internal/dto"
)

func TestCreateUserRequest_Validate(t *testing.T) {
    tests := []struct {
        name        string
        input       dto.CreateUserRequest
        wantErr     bool
        wantFields  []string // expected field names in validation error
    }{
        {
            name:    "valid request",
            input:   dto.CreateUserRequest{Username: "alice", Email: "alice@example.com", Password: "secret123"},
            wantErr: false,
        },
        {
            name:       "empty username",
            input:      dto.CreateUserRequest{Email: "alice@example.com", Password: "secret123"},
            wantErr:    true,
            wantFields: []string{"username"},
        },
        {
            name:       "invalid email",
            input:      dto.CreateUserRequest{Username: "alice", Email: "not-an-email", Password: "secret123"},
            wantErr:    true,
            wantFields: []string{"email"},
        },
        {
            name:       "short password",
            input:      dto.CreateUserRequest{Username: "alice", Email: "a@b.com", Password: "123"},
            wantErr:    true,
            wantFields: []string{"password"},
        },
        {
            name:       "multiple errors",
            input:      dto.CreateUserRequest{},
            wantErr:    true,
            wantFields: []string{"username", "email", "password"},
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            err := tc.input.Validate()

            if tc.wantErr && err == nil {
                t.Fatal("expected error but got nil")
            }
            if !tc.wantErr && err != nil {
                t.Fatalf("unexpected error: %v", err)
            }

            if tc.wantErr {
                appErr, ok := apperror.As(err)
                if !ok {
                    t.Fatalf("expected *apperror.AppError, got %T", err)
                }
                for _, field := range tc.wantFields {
                    if _, exists := appErr.Fields[field]; !exists {
                        t.Errorf("expected field %q in validation errors, got: %v", field, appErr.Fields)
                    }
                }
            }
        })
    }
}
```

---

## 13.4 Mocking with Interfaces

Go's implicit interface satisfaction makes mocking effortless — define an interface, write a struct that implements it.

```go
// internal/service/user_service.go uses this repository interface
type UserRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
    FindByEmail(ctx context.Context, email string) (*model.User, error)
    Create(ctx context.Context, s schema.CreateUser) (*model.User, error)
    Update(ctx context.Context, id uuid.UUID, s schema.UpdateUser) (*model.User, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

```go
// internal/service/mock/user_repository_mock.go
package mock

import (
    "context"

    "github.com/google/uuid"
    "myapp/internal/model"
    "myapp/internal/schema"
)

// UserRepositoryMock lets tests control exactly what the repository returns.
type UserRepositoryMock struct {
    FindByIDFn    func(ctx context.Context, id uuid.UUID) (*model.User, error)
    FindByEmailFn func(ctx context.Context, email string) (*model.User, error)
    CreateFn      func(ctx context.Context, s schema.CreateUser) (*model.User, error)
    UpdateFn      func(ctx context.Context, id uuid.UUID, s schema.UpdateUser) (*model.User, error)
    DeleteFn      func(ctx context.Context, id uuid.UUID) error
}

func (m *UserRepositoryMock) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
    return m.FindByIDFn(ctx, id)
}
func (m *UserRepositoryMock) FindByEmail(ctx context.Context, email string) (*model.User, error) {
    return m.FindByEmailFn(ctx, email)
}
func (m *UserRepositoryMock) Create(ctx context.Context, s schema.CreateUser) (*model.User, error) {
    return m.CreateFn(ctx, s)
}
func (m *UserRepositoryMock) Update(ctx context.Context, id uuid.UUID, s schema.UpdateUser) (*model.User, error) {
    return m.UpdateFn(ctx, id, s)
}
func (m *UserRepositoryMock) Delete(ctx context.Context, id uuid.UUID) error {
    return m.DeleteFn(ctx, id)
}
```

```go
// internal/service/user_service_test.go
package service_test

import (
    "context"
    "testing"
    "time"

    "github.com/google/uuid"
    "myapp/internal/apperror"
    "myapp/internal/dto"
    "myapp/internal/model"
    "myapp/internal/schema"
    "myapp/internal/service"
    "myapp/internal/service/mock"
)

func TestUserService_Create(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        repo := &mock.UserRepositoryMock{
            FindByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
                return nil, nil // email not taken
            },
            CreateFn: func(_ context.Context, s schema.CreateUser) (*model.User, error) {
                return &model.User{
                    Base:     model.Base{ID: uuid.New(), CreatedAt: time.Now()},
                    Username: s.Username,
                    Email:    s.Email,
                }, nil
            },
        }

        svc := service.NewUserService(repo)
        req := &dto.CreateUserRequest{
            Username: "alice",
            Email:    "alice@example.com",
            Password: "secret123",
        }

        user, err := svc.Create(context.Background(), req)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if user.Username != "alice" {
            t.Errorf("expected username alice, got %s", user.Username)
        }
    })

    t.Run("email already exists returns Conflict", func(t *testing.T) {
        repo := &mock.UserRepositoryMock{
            FindByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
                return &model.User{}, nil // email is taken
            },
        }

        svc := service.NewUserService(repo)
        req := &dto.CreateUserRequest{
            Username: "bob",
            Email:    "taken@example.com",
            Password: "secret123",
        }

        _, err := svc.Create(context.Background(), req)
        if err == nil {
            t.Fatal("expected conflict error, got nil")
        }
        if !apperror.Is(err, apperror.CodeConflict) {
            t.Errorf("expected CodeConflict, got: %v", err)
        }
    })
}
```

---

## 13.5 HTTP Handler Tests with `httptest`

No running server needed. `httptest.NewRecorder` captures the response.

```go
// internal/handler/user_handler_test.go
package handler_test

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/google/uuid"
    "myapp/internal/apperror"
    "myapp/internal/handler"
    "myapp/internal/model"
    "myapp/internal/schema"
)

// UserServiceMock implements the interface handler.UserService.
type UserServiceMock struct {
    CreateFn  func(ctx context.Context, s schema.CreateUser) (*model.User, error)
    FindByIDFn func(ctx context.Context, id uuid.UUID) (*model.User, error)
}

func (m *UserServiceMock) Create(ctx context.Context, s schema.CreateUser) (*model.User, error) {
    return m.CreateFn(ctx, s)
}
func (m *UserServiceMock) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
    return m.FindByIDFn(ctx, id)
}

// helper: create a handler wired to a mock service.
func newTestHandler(svc *UserServiceMock) *handler.Handler {
    return handler.New(svc, nil, newTestLogger())
}

func TestHandler_CreateUser(t *testing.T) {
    t.Run("201 on valid request", func(t *testing.T) {
        svc := &UserServiceMock{
            CreateFn: func(_ context.Context, s schema.CreateUser) (*model.User, error) {
                return &model.User{
                    Base:     model.Base{ID: uuid.New()},
                    Username: s.Username,
                    Email:    s.Email,
                }, nil
            },
        }

        body := `{"username":"alice","email":"alice@example.com","password":"secret123"}`
        req  := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
        req.Header.Set("Content-Type", "application/json")
        rec  := httptest.NewRecorder()

        newTestHandler(svc).CreateUser(rec, req)

        if rec.Code != http.StatusCreated {
            t.Errorf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
        }

        var resp map[string]any
        if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
            t.Fatalf("could not decode response: %v", err)
        }
        if resp["data"] == nil {
            t.Error("expected data in response envelope")
        }
    })

    t.Run("422 on invalid request", func(t *testing.T) {
        body := `{"username":"","email":"not-an-email","password":"123"}`
        req  := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
        req.Header.Set("Content-Type", "application/json")
        rec  := httptest.NewRecorder()

        newTestHandler(&UserServiceMock{}).CreateUser(rec, req)

        if rec.Code != http.StatusUnprocessableEntity {
            t.Errorf("expected 422, got %d", rec.Code)
        }

        var errResp handler.HTTPErrorResponse
        _ = json.NewDecoder(rec.Body).Decode(&errResp)
        if errResp.Code != string(apperror.CodeValidation) {
            t.Errorf("expected VALIDATION_ERROR, got %s", errResp.Code)
        }
        if errResp.Fields["email"] == "" {
            t.Error("expected field error for email")
        }
    })

    t.Run("409 when email already registered", func(t *testing.T) {
        svc := &UserServiceMock{
            CreateFn: func(_ context.Context, _ schema.CreateUser) (*model.User, error) {
                return nil, apperror.Conflict("email already registered")
            },
        }

        body := `{"username":"alice","email":"alice@example.com","password":"secret123"}`
        req  := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
        req.Header.Set("Content-Type", "application/json")
        rec  := httptest.NewRecorder()

        newTestHandler(svc).CreateUser(rec, req)

        if rec.Code != http.StatusConflict {
            t.Errorf("expected 409, got %d", rec.Code)
        }
    })
}
```

---

## 13.6 Integration Tests — Real PostgreSQL with Testcontainers

```bash
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
```

```go
// internal/testutil/db.go
package testutil

import (
    "context"
    "database/sql"
    "fmt"
    "testing"
    "time"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
    _ "github.com/jackc/pgx/v5/stdlib"
)

// NewTestDB spins up a PostgreSQL container, runs migrations, and returns
// a *sql.DB ready for integration tests. It is automatically torn down
// when the test ends (t.Cleanup).
func NewTestDB(t *testing.T) *sql.DB {
    t.Helper()
    ctx := context.Background()

    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(30*time.Second),
        ),
    )
    if err != nil {
        t.Fatalf("failed to start postgres container: %v", err)
    }

    t.Cleanup(func() {
        if err := pgContainer.Terminate(ctx); err != nil {
            t.Logf("failed to terminate container: %v", err)
        }
    })

    dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        t.Fatalf("failed to get connection string: %v", err)
    }

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        t.Fatalf("sql.Open: %v", err)
    }

    t.Cleanup(func() { db.Close() })

    // Apply migrations
    if err := runMigrations(db); err != nil {
        t.Fatalf("migrations failed: %v", err)
    }

    return db
}

func runMigrations(db *sql.DB) error {
    // Read migration files from disk and apply them in order.
    // For simplicity shown inline — in production use your migration runner.
    migrations := []string{
        `CREATE TABLE IF NOT EXISTS users (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            deleted_at TIMESTAMP DEFAULT NULL,
            active BOOLEAN NOT NULL DEFAULT TRUE,
            username VARCHAR(50) NOT NULL,
            password_hash VARCHAR(255) NOT NULL,
            email VARCHAR(255) NOT NULL UNIQUE,
            is_admin BOOLEAN NOT NULL DEFAULT FALSE
        )`,
        // ... add your other tables here
    }

    for _, m := range migrations {
        if _, err := db.Exec(m); err != nil {
            return fmt.Errorf("migration failed: %w", err)
        }
    }
    return nil
}
```

---

## 13.7 Full CRUD Integration Test

```go
// internal/repository/user_repository_test.go
package repository_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "myapp/internal/apperror"
    "myapp/internal/repository"
    "myapp/internal/schema"
    "myapp/internal/testutil"
)

func TestUserRepository_CRUD(t *testing.T) {
    db  := testutil.NewTestDB(t)
    repo := repository.NewUserRepository(db)
    ctx := context.Background()

    // ── CREATE ──────────────────────────────────────────────────────────────
    t.Run("Create", func(t *testing.T) {
        user, err := repo.Create(ctx, schema.CreateUser{
            Username:     "alice",
            Email:        "alice@example.com",
            PasswordHash: "hashed-password",
        })
        if err != nil {
            t.Fatalf("Create failed: %v", err)
        }
        if user.ID == uuid.Nil {
            t.Error("expected non-nil UUID from database")
        }
        if user.Username != "alice" {
            t.Errorf("expected username alice, got %s", user.Username)
        }
        if user.CreatedAt.IsZero() {
            t.Error("expected CreatedAt to be set by database")
        }
    })

    // ── FIND BY ID ───────────────────────────────────────────────────────────
    t.Run("FindByID", func(t *testing.T) {
        created, _ := repo.Create(ctx, schema.CreateUser{
            Username: "bob", Email: "bob@example.com", PasswordHash: "hash",
        })

        found, err := repo.FindByID(ctx, created.ID)
        if err != nil {
            t.Fatalf("FindByID failed: %v", err)
        }
        if found == nil {
            t.Fatal("expected user, got nil")
        }
        if found.Email != "bob@example.com" {
            t.Errorf("email mismatch: got %s", found.Email)
        }
    })

    t.Run("FindByID not found returns NotFound error", func(t *testing.T) {
        _, err := repo.FindByID(ctx, uuid.New())
        if err == nil {
            t.Fatal("expected error for unknown id")
        }
        if !apperror.Is(err, apperror.CodeNotFound) {
            t.Errorf("expected CodeNotFound, got: %v", err)
        }
    })

    // ── UPDATE ───────────────────────────────────────────────────────────────
    t.Run("Update", func(t *testing.T) {
        created, _ := repo.Create(ctx, schema.CreateUser{
            Username: "charlie", Email: "charlie@example.com", PasswordHash: "hash",
        })

        newName := "charlie-updated"
        updated, err := repo.Update(ctx, created.ID, schema.UpdateUser{Username: &newName})
        if err != nil {
            t.Fatalf("Update failed: %v", err)
        }
        if updated.Username != "charlie-updated" {
            t.Errorf("expected charlie-updated, got %s", updated.Username)
        }
        if !updated.UpdatedAt.After(created.UpdatedAt) {
            t.Error("expected UpdatedAt to be bumped after update")
        }
    })

    // ── FIND ALL ──────────────────────────────────────────────────────────────
    t.Run("FindAll returns paginated results", func(t *testing.T) {
        // DB already has rows from previous sub-tests (shared container in this test)
        result, err := repo.FindAll(ctx, repository.Page{Limit: 10, Offset: 0})
        if err != nil {
            t.Fatalf("FindAll failed: %v", err)
        }
        if result.TotalCount == 0 {
            t.Error("expected at least one user")
        }
        if len(result.Data) == 0 {
            t.Error("expected data slice to be populated")
        }
    })

    // ── DELETE (soft) ─────────────────────────────────────────────────────────
    t.Run("Delete soft-deletes the user", func(t *testing.T) {
        created, _ := repo.Create(ctx, schema.CreateUser{
            Username: "diana", Email: "diana@example.com", PasswordHash: "hash",
        })

        if err := repo.Delete(ctx, created.ID); err != nil {
            t.Fatalf("Delete failed: %v", err)
        }

        // User must be invisible after soft delete
        _, err := repo.FindByID(ctx, created.ID)
        if !apperror.Is(err, apperror.CodeNotFound) {
            t.Errorf("expected NotFound after soft delete, got: %v", err)
        }
    })

    t.Run("Delete non-existent user returns error", func(t *testing.T) {
        err := repo.Delete(ctx, uuid.New())
        if err == nil {
            t.Fatal("expected error deleting non-existent user")
        }
    })
}
```

---

## 13.8 Test Helpers & Fixtures

```go
// internal/testutil/fixtures.go
package testutil

import (
    "context"
    "testing"

    "myapp/internal/model"
    "myapp/internal/repository"
    "myapp/internal/schema"
)

// MustCreateUser creates a user and fails the test if it errors.
func MustCreateUser(t *testing.T, repo *repository.UserRepository, overrides ...func(*schema.CreateUser)) *model.User {
    t.Helper()

    s := schema.CreateUser{
        Username:     "testuser",
        Email:        "test@example.com",
        PasswordHash: "hash",
    }
    for _, o := range overrides {
        o(&s)
    }

    user, err := repo.Create(context.Background(), s)
    if err != nil {
        t.Fatalf("MustCreateUser: %v", err)
    }
    return user
}
```

```go
// Usage in tests — clean, readable
user := testutil.MustCreateUser(t, repo)

adminUser := testutil.MustCreateUser(t, repo, func(s *schema.CreateUser) {
    s.Username = "admin"
    s.Email    = "admin@example.com"
})
```

---

## 13.9 Middleware Testing

```go
// internal/middleware/logger_test.go
package middleware_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "myapp/internal/middleware"
)

func TestRequestIDMiddleware(t *testing.T) {
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := r.Context().Value(middleware.ContextKeyRequestID)
        if id == nil || id.(string) == "" {
            t.Error("expected request ID in context")
        }
        w.WriteHeader(http.StatusOK)
    })

    handler := middleware.RequestID(next)

    req := httptest.NewRequest(http.MethodGet, "/", nil)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rec.Code)
    }
    if rec.Header().Get("X-Request-ID") == "" {
        t.Error("expected X-Request-ID header in response")
    }
}
```

---

## 13.10 Sub-test Isolation with `t.Cleanup`

Each sub-test that touches the database should clean up after itself so tests are independent and order-agnostic.

```go
func TestZoneRepository(t *testing.T) {
    db   := testutil.NewTestDB(t)
    repo := repository.NewZoneRepository(db)

    t.Run("Create and find zone", func(t *testing.T) {
        // Seed a floor first
        floorID := testutil.MustCreateFloor(t, db)

        zone, err := repo.Create(context.Background(), schema.CreateZone{
            FloorID:   floorID,
            Name:      "Zone A",
            ShapeType: "polygon",
        })
        if err != nil {
            t.Fatalf("Create: %v", err)
        }

        // Cleanup: delete this zone so other sub-tests start clean
        t.Cleanup(func() {
            _, _ = db.Exec("DELETE FROM zones WHERE id = $1", zone.ID)
        })

        found, err := repo.FindByID(context.Background(), zone.ID)
        if err != nil {
            t.Fatalf("FindByID: %v", err)
        }
        if found.Name != "Zone A" {
            t.Errorf("expected Zone A, got %s", found.Name)
        }
    })
}
```

---

## 13.11 Running Tests

```bash
# All tests (unit + integration)
go test ./...

# Only unit tests (no containers needed — add a build tag)
go test -short ./...

# With race detector (always run in CI)
go test -race ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run a specific test
go test -run TestUserRepository_CRUD ./internal/repository/...

# Verbose output
go test -v -run TestUserService ./internal/service/...
```

### Skip integration tests in short mode

```go
func TestUserRepository_CRUD(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    // ...
}
```

---

## 13.12 Testing Checklist

| Layer | What to test | Technique |
|---|---|---|
| DTO / Validation | All valid/invalid input combinations | Table-driven unit tests |
| Service | Business rules, error conditions, edge cases | Mock repository, unit test |
| Repository | Real SQL queries, constraints, soft delete | Testcontainers + real PostgreSQL |
| Handler | Status codes, response shape, error propagation | `httptest`, mock service |
| Middleware | Context values, headers, next-handler delegation | `httptest` |
| Utilities | Pure functions (Map, Filter, etc.) | Simple unit tests |

---

*End of Go Developer Documentation Series*
