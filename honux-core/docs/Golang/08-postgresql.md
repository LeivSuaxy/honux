# 08 — PostgreSQL in Go (without ORM)

> Go's `database/sql` package provides a clean, low-level interface for SQL databases. Combined with `pgx` (the best PostgreSQL driver), you get full control, excellent performance, and no hidden magic.

---

## 8.1 Driver Setup

```bash
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/stdlib  # database/sql compatibility layer
```

```go
// cmd/api/main.go
import (
    "database/sql"
    _ "github.com/jackc/pgx/v5/stdlib"
)

func connectDB(dsn string) (*sql.DB, error) {
    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, fmt.Errorf("sql.Open: %w", err)
    }

    // Connection pool tuning
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(10)
    db.SetConnMaxLifetime(5 * time.Minute)
    db.SetConnMaxIdleTime(2 * time.Minute)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("db.Ping: %w", err)
    }
    return db, nil
}
```

### DSN format

```
postgres://user:password@host:5432/dbname?sslmode=disable
```

---

## 8.2 Migrations (without a heavy ORM)

Use `golang-migrate` — a standalone migration tool with no ORM attachment.

```bash
go get -tool github.com/golang-migrate/migrate/v4/cmd/migrate
```

```
migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── 000002_create_products.up.sql
└── 000002_create_products.down.sql
```

```sql
-- 000001_create_users.up.sql
CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL UNIQUE,
    role       TEXT        NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

```sql
-- 000001_create_users.down.sql
DROP TABLE IF EXISTS users;
```

Run migrations programmatically:

```go
import (
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func runMigrations(dsn string) error {
    m, err := migrate.New("file://migrations", dsn)
    if err != nil {
        return fmt.Errorf("migrate.New: %w", err)
    }
    defer m.Close()

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("migrate.Up: %w", err)
    }
    return nil
}
```

---

## 8.3 Query Patterns

### Single Row

```go
func (r *UserRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
    query := `
        SELECT id, name, email, role, created_at, updated_at
        FROM users
        WHERE id = $1
    `
    var u model.User
    err := r.db.QueryRowContext(ctx, query, id).
        Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)

    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil // caller decides if missing is an error
    }
    if err != nil {
        return nil, fmt.Errorf("FindByID(id=%d): %w", id, err)
    }
    return &u, nil
}
```

### Multiple Rows

```go
func (r *UserRepository) FindAll(ctx context.Context, filter UserFilter) ([]model.User, error) {
    query := `
        SELECT id, name, email, role, created_at, updated_at
        FROM users
        WHERE ($1::text IS NULL OR role = $1)
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `
    rows, err := r.db.QueryContext(ctx, query,
        nullString(filter.Role),
        filter.Limit,
        filter.Offset,
    )
    if err != nil {
        return nil, fmt.Errorf("FindAll: %w", err)
    }
    defer rows.Close()

    var users []model.User
    for rows.Next() {
        var u model.User
        if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
            return nil, fmt.Errorf("FindAll scan: %w", err)
        }
        users = append(users, u)
    }
    // Always check rows.Err() after iteration
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("FindAll rows: %w", err)
    }
    return users, nil
}

func nullString(s string) any {
    if s == "" {
        return nil
    }
    return s
}
```

### Insert Returning

```go
func (r *UserRepository) Create(ctx context.Context, req model.CreateUserRequest) (*model.User, error) {
    query := `
        INSERT INTO users (name, email, role)
        VALUES ($1, $2, $3)
        RETURNING id, name, email, role, created_at, updated_at
    `
    var u model.User
    err := r.db.QueryRowContext(ctx, query, req.Name, req.Email, req.Role).
        Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
    if err != nil {
        return nil, fmt.Errorf("Create: %w", err)
    }
    return &u, nil
}
```

### Update

```go
func (r *UserRepository) Update(ctx context.Context, id int64, req model.UpdateUserRequest) (*model.User, error) {
    query := `
        UPDATE users
        SET name = COALESCE($2, name),
            email = COALESCE($3, email),
            updated_at = NOW()
        WHERE id = $1
        RETURNING id, name, email, role, created_at, updated_at
    `
    var u model.User
    err := r.db.QueryRowContext(ctx, query, id,
        nullString(req.Name),
        nullString(req.Email),
    ).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)

    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("Update(id=%d): %w", id, err)
    }
    return &u, nil
}
```

### Delete

```go
func (r *UserRepository) Delete(ctx context.Context, id int64) (bool, error) {
    result, err := r.db.ExecContext(ctx,
        `DELETE FROM users WHERE id = $1`, id)
    if err != nil {
        return false, fmt.Errorf("Delete(id=%d): %w", id, err)
    }
    affected, _ := result.RowsAffected()
    return affected > 0, nil
}
```

---

## 8.4 Transactions

```go
func (r *UserRepository) TransferRole(ctx context.Context, fromID, toID int64, role string) error {
    tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    // defer rollback — if Commit() is called, Rollback() is a no-op
    defer tx.Rollback()

    // Step 1 — revoke from source
    _, err = tx.ExecContext(ctx,
        `UPDATE users SET role = 'user' WHERE id = $1`, fromID)
    if err != nil {
        return fmt.Errorf("revoke role: %w", err)
    }

    // Step 2 — assign to target
    _, err = tx.ExecContext(ctx,
        `UPDATE users SET role = $1 WHERE id = $2`, role, toID)
    if err != nil {
        return fmt.Errorf("assign role: %w", err)
    }

    return tx.Commit()
}
```

### Transaction Helper

Wrap the boilerplate in a reusable helper:

```go
func withTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    if err := fn(tx); err != nil {
        return err
    }
    return tx.Commit()
}

// Usage
err := withTx(ctx, db, func(tx *sql.Tx) error {
    _, err := tx.ExecContext(ctx, `UPDATE accounts SET balance = balance - $1 WHERE id = $2`, 100, fromID)
    if err != nil {
        return err
    }
    _, err = tx.ExecContext(ctx, `UPDATE accounts SET balance = balance + $1 WHERE id = $2`, 100, toID)
    return err
})
```

---

## 8.5 Prepared Statements

Use prepared statements for queries that run many times — they improve performance and resist SQL injection.

```go
type UserRepository struct {
    db         *sql.DB
    stmtFindID *sql.Stmt
}

func NewUserRepository(ctx context.Context, db *sql.DB) (*UserRepository, error) {
    stmt, err := db.PrepareContext(ctx,
        `SELECT id, name, email FROM users WHERE id = $1`)
    if err != nil {
        return nil, fmt.Errorf("prepare FindByID: %w", err)
    }
    return &UserRepository{db: db, stmtFindID: stmt}, nil
}

func (r *UserRepository) Close() {
    r.stmtFindID.Close()
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
    var u model.User
    err := r.stmtFindID.QueryRowContext(ctx, id).
        Scan(&u.ID, &u.Name, &u.Email)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil
    }
    return &u, err
}
```

---

## 8.6 Batch Inserts

Use `UNNEST` for efficient bulk inserts — one round-trip for many rows.

```go
import "github.com/lib/pq"

func (r *UserRepository) BulkInsert(ctx context.Context, users []model.CreateUserRequest) error {
    if len(users) == 0 {
        return nil
    }

    names  := make([]string, len(users))
    emails := make([]string, len(users))
    for i, u := range users {
        names[i]  = u.Name
        emails[i] = u.Email
    }

    _, err := r.db.ExecContext(ctx, `
        INSERT INTO users (name, email)
        SELECT * FROM UNNEST($1::text[], $2::text[])
        ON CONFLICT (email) DO NOTHING
    `, pq.Array(names), pq.Array(emails))

    return err
}
```

---

## 8.7 Handling NULL Values

```go
import "database/sql"

type UserRow struct {
    ID        int64
    Name      string
    Bio       sql.NullString    // nullable text column
    Score     sql.NullFloat64   // nullable numeric
    DeletedAt sql.NullTime      // nullable timestamp
}

func (r *UserRepository) FindWithNulls(ctx context.Context, id int64) (*UserRow, error) {
    var u UserRow
    err := r.db.QueryRowContext(ctx,
        `SELECT id, name, bio, score, deleted_at FROM users WHERE id = $1`, id,
    ).Scan(&u.ID, &u.Name, &u.Bio, &u.Score, &u.DeletedAt)
    if err != nil {
        return nil, err
    }
    return &u, nil
}

// Reading nullable values
if u.Bio.Valid {
    fmt.Println("bio:", u.Bio.String)
}
```

---

## 8.8 pgx Native Pool (High Performance)

For maximum performance, use `pgx` directly (bypassing `database/sql`):

```go
import "github.com/jackc/pgx/v5/pgxpool"

func connectPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
    cfg, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, err
    }
    cfg.MaxConns = 25
    cfg.MinConns = 5
    cfg.MaxConnLifetime = 5 * time.Minute

    pool, err := pgxpool.NewWithConfig(ctx, cfg)
    if err != nil {
        return nil, err
    }
    return pool, pool.Ping(ctx)
}

// Use pool
func findUser(ctx context.Context, pool *pgxpool.Pool, id int64) (*model.User, error) {
    var u model.User
    err := pool.QueryRow(ctx,
        `SELECT id, name, email FROM users WHERE id = $1`, id,
    ).Scan(&u.ID, &u.Name, &u.Email)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, nil
    }
    return &u, err
}
```

---

## 8.9 Repository Interface Pattern

Decouple the service layer from the concrete repository to allow testing with mocks:

```go
// internal/repository/interfaces.go
package repository

type UserRepo interface {
    FindAll(ctx context.Context, filter UserFilter) ([]model.User, error)
    FindByID(ctx context.Context, id int64) (*model.User, error)
    Create(ctx context.Context, req model.CreateUserRequest) (*model.User, error)
    Update(ctx context.Context, id int64, req model.UpdateUserRequest) (*model.User, error)
    Delete(ctx context.Context, id int64) (bool, error)
}

// internal/repository/mock/user.go — for unit tests
type MockUserRepo struct {
    Users map[int64]model.User
    NextID int64
}

func (m *MockUserRepo) FindByID(ctx context.Context, id int64) (*model.User, error) {
    u, ok := m.Users[id]
    if !ok {
        return nil, nil
    }
    return &u, nil
}
// ... implement other methods
```

---

## 8.10 Error Handling for Postgres Errors

```go
import "github.com/jackc/pgx/v5/pgconn"

func isDuplicateKeyError(err error) bool {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return pgErr.Code == "23505" // unique_violation
    }
    return false
}

func (r *UserRepository) Create(ctx context.Context, req model.CreateUserRequest) (*model.User, error) {
    u, err := r.create(ctx, req)
    if err != nil {
        if isDuplicateKeyError(err) {
            return nil, fmt.Errorf("email %q already exists", req.Email)
        }
        return nil, err
    }
    return u, nil
}
```

Common PostgreSQL error codes:

| Code | Name | Meaning |
|---|---|---|
| `23505` | `unique_violation` | Duplicate key |
| `23503` | `foreign_key_violation` | FK constraint failed |
| `23502` | `not_null_violation` | NOT NULL constraint |
| `40001` | `serialization_failure` | Retry-able transaction conflict |
| `42P01` | `undefined_table` | Table doesn't exist |

---

*Next: [09 — Event-Driven Architecture →](./09-event-driven-architecture.md)*
