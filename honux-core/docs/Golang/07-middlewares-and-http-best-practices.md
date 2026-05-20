# 07 — Middlewares, Interceptors & HTTP Components

> Middleware in Go is simply a function that wraps an `http.Handler` and returns an `http.Handler`. No framework magic — just function composition.

---

## 7.1 The Middleware Signature

```go
type Middleware func(http.Handler) http.Handler
```

A middleware intercepts the request, optionally does work before/after calling the next handler:

```go
func ExampleMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // --- before ---
        fmt.Println("before handler")

        next.ServeHTTP(w, r) // call the next handler in the chain

        // --- after ---
        fmt.Println("after handler")
    })
}
```

---

## 7.2 Chaining Middleware

```go
// internal/middleware/chain.go
package middleware

import "net/http"

// Chain applies middleware in left-to-right order.
// Chain(A, B, C)(handler) → A(B(C(handler)))
// Request flows: A → B → C → handler → C → B → A
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
    return func(final http.Handler) http.Handler {
        for i := len(middlewares) - 1; i >= 0; i-- {
            final = middlewares[i](final)
        }
        return final
    }
}

// Usage
stack := middleware.Chain(
    middleware.Recover,
    middleware.RequestID,
    middleware.Logger,
    middleware.CORS,
)
mux.Handle("/", stack(myHandler))
```

---

## 7.3 Request ID Middleware

Assigns a unique ID to every request — essential for distributed tracing and log correlation.

```go
// internal/middleware/request_id.go
package middleware

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "net/http"
)

type contextKey string

const RequestIDKey contextKey = "requestID"

func RequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := r.Header.Get("X-Request-ID")
        if id == "" {
            b := make([]byte, 8)
            rand.Read(b)
            id = hex.EncodeToString(b)
        }
        ctx := context.WithValue(r.Context(), RequestIDKey, id)
        w.Header().Set("X-Request-ID", id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Helper to extract it anywhere downstream
func GetRequestID(ctx context.Context) string {
    id, _ := ctx.Value(RequestIDKey).(string)
    return id
}
```

---

## 7.4 Structured Logger Middleware

Uses Go 1.21's `log/slog` for structured JSON logging.

```go
// internal/middleware/logger.go
package middleware

import (
    "log/slog"
    "net/http"
    "time"
)

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
    http.ResponseWriter
    status      int
    wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
    return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
    if rw.wroteHeader {
        return
    }
    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
    rw.wroteHeader = true
}

func Logger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        wrapped := wrapResponseWriter(w)

        next.ServeHTTP(wrapped, r)

        slog.Info("request",
            "method",     r.Method,
            "path",       r.URL.Path,
            "status",     wrapped.status,
            "duration_ms", time.Since(start).Milliseconds(),
            "remote_addr", r.RemoteAddr,
            "request_id", GetRequestID(r.Context()),
        )
    })
}
```

---

## 7.5 Panic Recovery Middleware

Recovers from panics inside handlers and returns a 500 instead of crashing the server.

```go
// internal/middleware/recover.go
package middleware

import (
    "log/slog"
    "net/http"
    "runtime/debug"
)

func Recover(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                slog.Error("panic recovered",
                    "error",      rec,
                    "stack",      string(debug.Stack()),
                    "request_id", GetRequestID(r.Context()),
                )
                http.Error(w, "internal server error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

---

## 7.6 CORS Middleware

```go
// internal/middleware/cors.go
package middleware

import "net/http"

type CORSConfig struct {
    AllowedOrigins []string
    AllowedMethods []string
    AllowedHeaders []string
}

func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
    allowed := make(map[string]bool)
    for _, o := range cfg.AllowedOrigins {
        allowed[o] = true
    }

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            if allowed[origin] || allowed["*"] {
                w.Header().Set("Access-Control-Allow-Origin",  origin)
                w.Header().Set("Access-Control-Allow-Methods", join(cfg.AllowedMethods))
                w.Header().Set("Access-Control-Allow-Headers", join(cfg.AllowedHeaders))
            }
            // Handle preflight
            if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusNoContent)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

func join(ss []string) string {
    result := ""
    for i, s := range ss {
        if i > 0 { result += ", " }
        result += s
    }
    return result
}
```

---

## 7.7 Authentication Middleware (JWT)

```go
// internal/middleware/auth.go
package middleware

import (
    "context"
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v5"
)

type contextKey string
const UserClaimsKey contextKey = "userClaims"

type Claims struct {
    UserID int64  `json:"user_id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

func Auth(secret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authHeader := r.Header.Get("Authorization")
            if !strings.HasPrefix(authHeader, "Bearer ") {
                http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
                return
            }

            tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
            claims := &Claims{}

            token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
                if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                    return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
                }
                return []byte(secret), nil
            })

            if err != nil || !token.Valid {
                http.Error(w, "invalid or expired token", http.StatusUnauthorized)
                return
            }

            ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func GetClaims(ctx context.Context) *Claims {
    c, _ := ctx.Value(UserClaimsKey).(*Claims)
    return c
}

// Role-based authorization
func RequireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := GetClaims(r.Context())
            if claims == nil || claims.Role != role {
                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 7.8 Rate Limiting Middleware

```go
// internal/middleware/ratelimit.go
package middleware

import (
    "net/http"
    "sync"
    "time"
)

type rateLimiter struct {
    mu       sync.Mutex
    requests map[string][]time.Time
    limit    int
    window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
    return &rateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (rl *rateLimiter) allow(ip string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-rl.window)

    // Remove timestamps outside the window
    valid := rl.requests[ip][:0]
    for _, t := range rl.requests[ip] {
        if t.After(cutoff) {
            valid = append(valid, t)
        }
    }
    rl.requests[ip] = valid

    if len(valid) >= rl.limit {
        return false
    }
    rl.requests[ip] = append(rl.requests[ip], now)
    return true
}

func RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
    rl := newRateLimiter(limit, window)
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := r.RemoteAddr
            if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
                ip = strings.Split(fwd, ",")[0]
            }
            if !rl.allow(ip) {
                w.Header().Set("Retry-After", "60")
                http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

> For production, use `golang.org/x/time/rate` (token bucket) or a Redis-backed limiter for multi-instance deployments.

---

## 7.9 Request Timeout Middleware

```go
func Timeout(d time.Duration) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx, cancel := context.WithTimeout(r.Context(), d)
            defer cancel()
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

---

## 7.10 Route-Specific Middleware

Apply middleware to a specific route by wrapping the handler directly:

```go
// Global middleware on the mux
globalStack := middleware.Chain(middleware.Recover, middleware.RequestID, middleware.Logger)
mux.Handle("/", globalStack(mainHandler))

// Auth only on protected routes
authMiddleware := middleware.Auth(os.Getenv("JWT_SECRET"))

mux.Handle("GET /api/v1/profile",
    globalStack(authMiddleware(http.HandlerFunc(profileHandler.Get))),
)

mux.Handle("POST /api/v1/admin/users",
    globalStack(
        authMiddleware(
            middleware.RequireRole("admin")(
                http.HandlerFunc(adminHandler.CreateUser),
            ),
        ),
    ),
)
```

---

## 7.11 Best Practices for HTTP Applications

### Configuration

```go
// Load from environment — never hardcode secrets
type Config struct {
    Port        int
    DatabaseURL string
    JWTSecret   string
    LogLevel    string
    Environment string
}

func LoadConfig() Config {
    return Config{
        Port:        getEnvInt("PORT", 8080),
        DatabaseURL: mustGetEnv("DATABASE_URL"),
        JWTSecret:   mustGetEnv("JWT_SECRET"),
        LogLevel:    getEnv("LOG_LEVEL", "info"),
        Environment: getEnv("ENV", "development"),
    }
}

func mustGetEnv(key string) string {
    v := os.Getenv(key)
    if v == "" {
        log.Fatalf("required environment variable %q is not set", key)
    }
    return v
}
```

### Error Handling — Sentinel Errors

```go
// internal/apierror/apierror.go
package apierror

import (
    "errors"
    "net/http"
)

type APIError struct {
    Status  int
    Message string
    Err     error
}

func (e *APIError) Error() string { return e.Message }
func (e *APIError) Unwrap() error { return e.Err }

var (
    ErrNotFound   = &APIError{Status: http.StatusNotFound,   Message: "resource not found"}
    ErrBadRequest = &APIError{Status: http.StatusBadRequest, Message: "bad request"}
    ErrForbidden  = &APIError{Status: http.StatusForbidden,  Message: "forbidden"}
)

func HandleError(w http.ResponseWriter, err error) {
    var apiErr *APIError
    if errors.As(err, &apiErr) {
        writeJSON(w, apiErr.Status, map[string]string{"error": apiErr.Message})
        return
    }
    writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}
```

### Security Checklist

```go
func secureHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options",    "nosniff")
        w.Header().Set("X-Frame-Options",           "DENY")
        w.Header().Set("X-XSS-Protection",          "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
        w.Header().Set("Referrer-Policy",           "strict-origin-when-cross-origin")
        next.ServeHTTP(w, r)
    })
}
```

### Body Size Limit

```go
func MaxBodySize(limit int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, limit)
            next.ServeHTTP(w, r)
        })
    }
}

// Usage: 1 MB limit
mux.Handle("/upload", MaxBodySize(1<<20)(http.HandlerFunc(uploadHandler)))
```

### Complete Middleware Stack (Recommended Order)

```go
stack := middleware.Chain(
    middleware.Recover,       // 1. Always first — catch panics
    middleware.RequestID,     // 2. Tag every request
    middleware.Logger,        // 3. Log with request ID
    secureHeaders,            // 4. Set security headers
    middleware.CORS(corsConfig),  // 5. Handle CORS
    middleware.RateLimit(100, time.Minute), // 6. Rate limit
    middleware.Timeout(30 * time.Second),   // 7. Enforce timeout
    // Auth is applied per-route, not globally
)
```

---

*Next: [08 — PostgreSQL without ORM →](./08-postgresql.md)*
