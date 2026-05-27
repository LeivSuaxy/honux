# 11 — Repository Relations & Optimal SQL

> The key question when loading related entities is: **how many queries should this cost?** This guide covers every practical pattern — from simple eager loading to batch fetching — staying entirely within `database/sql` and raw SQL.

---

## 11.1 The N+1 Problem (and Why to Avoid It)

The naive approach — load a parent, then query for each child in a loop — is the N+1 problem.

```go
// ❌ N+1: 1 query for floors + N queries for zones
floors, _ := repo.FindAllFloors(ctx)
for _, f := range floors {
    f.Zones, _ = repo.FindZonesByFloorID(ctx, f.ID) // N extra queries
}
```

For 100 floors that's 101 queries. The solutions are **JOINs** for small related sets and **batch IN queries** for large sets.

---

## 11.2 Pattern 1 — Eager Load with JOIN (one query)

Best for: loading a parent **with a few related children** where you always need both.

```go
// internal/model/models.go (extended)
type Zone struct {
    Base
    FloorID         uuid.UUID       `json:"floor_id"`
    Name            string          `json:"name"`
    ShortIdentifier *string         `json:"short_identifier,omitempty"`
    ShapeType       string          `json:"shape_type"`
    Geometry        json.RawMessage `json:"geometry,omitempty"`
    Color           *string         `json:"color,omitempty"`

    // Eager-loaded relation
    Floor *Floor `json:"floor,omitempty"`
}
```

```go
// internal/repository/zone_repository.go
package repository

import (
    "context"
    "database/sql"
    "errors"
    "fmt"

    "github.com/google/uuid"
    "myapp/internal/apperror"
    "myapp/internal/model"
)

type ZoneRepository struct{ db *sql.DB }

func NewZoneRepository(db *sql.DB) *ZoneRepository {
    return &ZoneRepository{db: db}
}

// FindByIDWithFloor loads a Zone and its Floor in a single JOIN query.
func (r *ZoneRepository) FindByIDWithFloor(ctx context.Context, id uuid.UUID) (*model.Zone, error) {
    query := `
        SELECT
            z.id, z.created_at, z.updated_at, z.deleted_at, z.active,
            z.floor_id, z.name, z.short_identifier, z.shape_type, z.geometry, z.color,
            f.id, f.created_at, f.updated_at, f.deleted_at, f.active,
            f.name, f.level
        FROM zones z
        INNER JOIN floors f ON f.id = z.floor_id
        WHERE z.id = $1
          AND z.deleted_at IS NULL
          AND f.deleted_at IS NULL
    `

    var (
        z          model.Zone
        f          model.Floor
        zDeletedAt sql.NullTime
        fDeletedAt sql.NullTime
        shortID    sql.NullString
        color      sql.NullString
        geometry   []byte // JSONB comes back as []byte
    )

    err := r.db.QueryRowContext(ctx, query, id).Scan(
        // zone columns
        &z.ID, &z.CreatedAt, &z.UpdatedAt, &zDeletedAt, &z.Active,
        &z.FloorID, &z.Name, &shortID, &z.ShapeType, &geometry, &color,
        // floor columns
        &f.ID, &f.CreatedAt, &f.UpdatedAt, &fDeletedAt, &f.Active,
        &f.Name, &f.Level,
    )
    if errors.Is(err, sql.ErrNoRows) {
        return nil, apperror.NotFound("zone")
    }
    if err != nil {
        return nil, fmt.Errorf("ZoneRepository.FindByIDWithFloor: %w", apperror.Internal(err))
    }

    // Map nullables
    if zDeletedAt.Valid { z.DeletedAt = &zDeletedAt.Time }
    if fDeletedAt.Valid { f.DeletedAt = &fDeletedAt.Time }
    if shortID.Valid    { z.ShortIdentifier = &shortID.String }
    if color.Valid      { z.Color = &color.String }
    if len(geometry) > 0 { z.Geometry = geometry }

    // Attach relation
    z.Floor = &f

    return &z, nil
}
```

---

## 11.3 Pattern 2 — Load Parent + Multiple Children (one-to-many)

Loading a Floor with all its Zones in one query. Multiple rows are returned; you group them manually.

```go
type FloorWithZones struct {
    model.Floor
    Zones []*model.Zone `json:"zones"`
}

// FindFloorWithZones returns a floor and all its active zones — single query.
func (r *FloorRepository) FindFloorWithZones(ctx context.Context, id uuid.UUID) (*FloorWithZones, error) {
    query := `
        SELECT
            f.id, f.created_at, f.updated_at, f.deleted_at, f.active,
            f.name, f.level,
            z.id, z.created_at, z.updated_at, z.deleted_at, z.active,
            z.name, z.short_identifier, z.shape_type, z.geometry, z.color
        FROM floors f
        LEFT JOIN zones z
               ON z.floor_id = f.id
              AND z.deleted_at IS NULL
        WHERE f.id = $1
          AND f.deleted_at IS NULL
        ORDER BY z.name
    `

    rows, err := r.db.QueryContext(ctx, query, id)
    if err != nil {
        return nil, fmt.Errorf("FindFloorWithZones: %w", apperror.Internal(err))
    }
    defer rows.Close()

    var result *FloorWithZones

    for rows.Next() {
        var (
            f          model.Floor
            fDeletedAt sql.NullTime
            // Zone fields — nullable because LEFT JOIN may have no zones
            zID        uuid.NullUUID
            zCreatedAt sql.NullTime
            zUpdatedAt sql.NullTime
            zDeletedAt sql.NullTime
            zActive    sql.NullBool
            zName      sql.NullString
            zShortID   sql.NullString
            zShapeType sql.NullString
            zGeometry  []byte
            zColor     sql.NullString
        )

        if err := rows.Scan(
            &f.ID, &f.CreatedAt, &f.UpdatedAt, &fDeletedAt, &f.Active, &f.Name, &f.Level,
            &zID, &zCreatedAt, &zUpdatedAt, &zDeletedAt, &zActive,
            &zName, &zShortID, &zShapeType, &zGeometry, &zColor,
        ); err != nil {
            return nil, fmt.Errorf("FindFloorWithZones scan: %w", apperror.Internal(err))
        }

        // First row initialises the parent
        if result == nil {
            if fDeletedAt.Valid { f.DeletedAt = &fDeletedAt.Time }
            result = &FloorWithZones{Floor: f, Zones: []*model.Zone{}}
        }

        // LEFT JOIN row may have no zone (floor with zero zones)
        if !zID.Valid {
            continue
        }

        z := &model.Zone{}
        z.ID       = zID.UUID
        z.FloorID  = f.ID
        if zCreatedAt.Valid { z.CreatedAt = zCreatedAt.Time }
        if zUpdatedAt.Valid { z.UpdatedAt = zUpdatedAt.Time }
        if zDeletedAt.Valid { z.DeletedAt = &zDeletedAt.Time }
        if zActive.Valid    { z.Active    = zActive.Bool }
        if zName.Valid      { z.Name      = zName.String }
        if zShortID.Valid   { z.ShortIdentifier = &zShortID.String }
        if zShapeType.Valid { z.ShapeType = zShapeType.String }
        if zColor.Valid     { z.Color     = &zColor.String }
        if len(zGeometry) > 0 { z.Geometry = zGeometry }

        result.Zones = append(result.Zones, z)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("FindFloorWithZones rows: %w", apperror.Internal(err))
    }
    if result == nil {
        return nil, apperror.NotFound("floor")
    }

    return result, nil
}
```

---

## 11.4 Pattern 3 — Batch IN Query (avoid N+1 at scale)

When loading related data for a **list** of parents, use a single `IN` query instead of N queries.

```go
// internal/repository/controller_repository.go

// FindByZoneIDs loads all controllers for a slice of zone IDs — single query.
func (r *ControllerRepository) FindByZoneIDs(
    ctx context.Context,
    zoneIDs []uuid.UUID,
) (map[uuid.UUID][]*model.Controller, error) {

    if len(zoneIDs) == 0 {
        return nil, nil
    }

    // Build $1,$2,$3... placeholders dynamically
    placeholders, args := buildInPlaceholders(zoneIDs)

    query := fmt.Sprintf(`
        SELECT id, created_at, updated_at, deleted_at, active,
               zone_id, induced_id, name, description, device_type,
               last_ip_address, mqtt_topic, is_online, last_ping, pos_x, pos_y
        FROM controllers
        WHERE zone_id IN (%s)
          AND deleted_at IS NULL
        ORDER BY name
    `, placeholders)

    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("FindByZoneIDs: %w", apperror.Internal(err))
    }
    defer rows.Close()

    result := make(map[uuid.UUID][]*model.Controller)

    for rows.Next() {
        c, err := scanController(rows)
        if err != nil {
            return nil, err
        }
        result[c.ZoneID] = append(result[c.ZoneID], c)
    }

    return result, rows.Err()
}

// buildInPlaceholders builds "$1,$2,$3" and a matching []any args slice.
func buildInPlaceholders[T any](items []T) (string, []any) {
    args := make([]any, len(items))
    parts := make([]string, len(items))
    for i, v := range items {
        args[i] = v
        parts[i] = fmt.Sprintf("$%d", i+1)
    }
    return strings.Join(parts, ","), args
}
```

### Usage in service — 2 queries total regardless of list size

```go
func (s *ZoneService) ListFloorsWithControllers(ctx context.Context) ([]*FloorWithControllers, error) {
    // Query 1: get all zones
    zones, err := s.zoneRepo.FindAll(ctx)
    if err != nil {
        return nil, err
    }

    // Collect zone IDs
    zoneIDs := make([]uuid.UUID, len(zones))
    for i, z := range zones {
        zoneIDs[i] = z.ID
    }

    // Query 2: get all controllers for those zones at once
    controllersByZone, err := s.ctrlRepo.FindByZoneIDs(ctx, zoneIDs)
    if err != nil {
        return nil, err
    }

    // Assemble in-memory
    result := make([]*FloorWithControllers, len(zones))
    for i, z := range zones {
        result[i] = &FloorWithControllers{
            Zone:        z,
            Controllers: controllersByZone[z.ID], // nil if no controllers
        }
    }
    return result, nil
}
```

---

## 11.5 Pattern 4 — Optional Relations via a `With` Options Pattern

Let callers decide what to load without duplicating repository methods.

```go
// internal/repository/options.go
package repository

type QueryOptions struct {
    WithFloor       bool
    WithControllers bool
    WithComponents  bool
}

type Option func(*QueryOptions)

func WithFloor() Option       { return func(o *QueryOptions) { o.WithFloor = true } }
func WithControllers() Option { return func(o *QueryOptions) { o.WithControllers = true } }
func WithComponents() Option  { return func(o *QueryOptions) { o.WithComponents = true } }
```

```go
// FindByID respects the caller's requested relations.
func (r *ZoneRepository) FindByID(
    ctx context.Context,
    id uuid.UUID,
    opts ...Option,
) (*model.Zone, error) {

    o := &QueryOptions{}
    for _, opt := range opts {
        opt(o)
    }

    // Base query — no joins
    zone, err := r.findByIDBase(ctx, id)
    if err != nil || zone == nil {
        return zone, err
    }

    if o.WithFloor {
        floor, err := r.floorRepo.FindByID(ctx, zone.FloorID)
        if err != nil {
            return nil, fmt.Errorf("ZoneRepository.FindByID load floor: %w", err)
        }
        zone.Floor = floor
    }

    if o.WithControllers {
        ctrls, err := r.ctrlRepo.FindByZoneIDs(ctx, []uuid.UUID{zone.ID})
        if err != nil {
            return nil, fmt.Errorf("ZoneRepository.FindByID load controllers: %w", err)
        }
        zone.Controllers = ctrls[zone.ID]
    }

    return zone, nil
}

// Caller controls what gets loaded — no magic
zone, err := repo.FindByID(ctx, id)                                     // bare
zone, err := repo.FindByID(ctx, id, repository.WithFloor())             // + floor
zone, err := repo.FindByID(ctx, id, repository.WithFloor(),
                                    repository.WithControllers())       // + both
```

---

## 11.6 INSERT and RETURNING

Always use `RETURNING` to populate the auto-generated fields after an insert. This avoids a second SELECT.

```go
// Create a Zone and return it fully populated (id, timestamps, etc.)
func (r *ZoneRepository) Create(ctx context.Context, z *model.Zone) (*model.Zone, error) {
    query := `
        INSERT INTO zones (floor_id, name, short_identifier, shape_type, geometry, color)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, created_at, updated_at, active
    `

    var result model.Zone
    result.FloorID        = z.FloorID
    result.Name           = z.Name
    result.ShortIdentifier = z.ShortIdentifier
    result.ShapeType      = z.ShapeType
    result.Geometry       = z.Geometry
    result.Color          = z.Color

    err := r.db.QueryRowContext(ctx, query,
        z.FloorID,
        z.Name,
        nullableString(z.ShortIdentifier), // helper: *string → sql.NullString
        z.ShapeType,
        nullableBytes(z.Geometry),
        nullableString(z.Color),
    ).Scan(&result.ID, &result.CreatedAt, &result.UpdatedAt, &result.Active)

    if err != nil {
        if isForeignKeyViolation(err) {
            return nil, apperror.NotFound("floor")
        }
        return nil, fmt.Errorf("ZoneRepository.Create: %w", apperror.Internal(err))
    }

    return &result, nil
}

// ── Nullable helpers (write side) ────────────────────────────────────────────

func nullableString(s *string) sql.NullString {
    if s == nil {
        return sql.NullString{}
    }
    return sql.NullString{String: *s, Valid: true}
}

func nullableBytes(b []byte) any {
    if len(b) == 0 {
        return nil
    }
    return b
}
```

---

## 11.7 Bulk INSERT

```go
// BulkCreateComponentLogs inserts many logs in a single statement.
func (r *ComponentLogRepository) BulkCreate(
    ctx context.Context,
    logs []*model.ComponentLog,
) error {
    if len(logs) == 0 {
        return nil
    }

    // Build multi-row VALUES clause: ($1,$2,$3), ($4,$5,$6), ...
    const colsPerRow = 5
    args := make([]any, 0, len(logs)*colsPerRow)
    valueStrings := make([]string, 0, len(logs))

    for i, l := range logs {
        base := i * colsPerRow
        valueStrings = append(valueStrings,
            fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)",
                base+1, base+2, base+3, base+4, base+5),
        )
        args = append(args,
            l.ComponentID,
            l.Type,
            l.Value,
            nullableString(l.Unit),
            nullableBytes(l.Metadata),
        )
    }

    query := fmt.Sprintf(`
        INSERT INTO component_logs (component_id, type, value, unit, metadata)
        VALUES %s
    `, strings.Join(valueStrings, ", "))

    _, err := r.db.ExecContext(ctx, query, args...)
    if err != nil {
        return fmt.Errorf("ComponentLogRepository.BulkCreate: %w", apperror.Internal(err))
    }
    return nil
}
```

---

## 11.8 Pagination

```go
// internal/repository/pagination.go
package repository

type Page struct {
    Limit  int `json:"limit"`
    Offset int `json:"offset"`
}

func (p Page) Sanitise() Page {
    if p.Limit <= 0 || p.Limit > 100 {
        p.Limit = 20
    }
    if p.Offset < 0 {
        p.Offset = 0
    }
    return p
}

type PagedResult[T any] struct {
    Data       []T `json:"data"`
    TotalCount int `json:"total_count"`
    Limit      int `json:"limit"`
    Offset     int `json:"offset"`
}
```

```go
func (r *ZoneRepository) FindAll(
    ctx context.Context,
    floorID uuid.UUID,
    page repository.Page,
) (*repository.PagedResult[*model.Zone], error) {
    page = page.Sanitise()

    countQuery := `SELECT COUNT(*) FROM zones WHERE floor_id = $1 AND deleted_at IS NULL`
    var total int
    if err := r.db.QueryRowContext(ctx, countQuery, floorID).Scan(&total); err != nil {
        return nil, fmt.Errorf("ZoneRepository.FindAll count: %w", apperror.Internal(err))
    }

    query := `
        SELECT id, created_at, updated_at, deleted_at, active,
               floor_id, name, short_identifier, shape_type, geometry, color
        FROM zones
        WHERE floor_id = $1
          AND deleted_at IS NULL
        ORDER BY name
        LIMIT $2 OFFSET $3
    `
    rows, err := r.db.QueryContext(ctx, query, floorID, page.Limit, page.Offset)
    if err != nil {
        return nil, fmt.Errorf("ZoneRepository.FindAll: %w", apperror.Internal(err))
    }
    defer rows.Close()

    zones := make([]*model.Zone, 0, page.Limit)
    for rows.Next() {
        z, err := scanZone(rows)
        if err != nil {
            return nil, err
        }
        zones = append(zones, z)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("ZoneRepository.FindAll rows: %w", apperror.Internal(err))
    }

    return &repository.PagedResult[*model.Zone]{
        Data:       zones,
        TotalCount: total,
        Limit:      page.Limit,
        Offset:     page.Offset,
    }, nil
}
```

---

## 11.9 Reusable Scan Helper

Extract scan logic into a private function so every method reuses the same mapping:

```go
// scanZone scans a single zone row from any *sql.Row or *sql.Rows.
func scanZone(s interface {
    Scan(dest ...any) error
}) (*model.Zone, error) {
    var (
        z          model.Zone
        deletedAt  sql.NullTime
        shortID    sql.NullString
        geometry   []byte
        color      sql.NullString
    )

    if err := s.Scan(
        &z.ID, &z.CreatedAt, &z.UpdatedAt, &deletedAt, &z.Active,
        &z.FloorID, &z.Name, &shortID, &z.ShapeType, &geometry, &color,
    ); err != nil {
        return nil, fmt.Errorf("scanZone: %w", apperror.Internal(err))
    }

    if deletedAt.Valid { z.DeletedAt = &deletedAt.Time }
    if shortID.Valid   { z.ShortIdentifier = &shortID.String }
    if color.Valid     { z.Color = &color.String }
    if len(geometry) > 0 { z.Geometry = geometry }

    return &z, nil
}
```

---

## 11.10 Pattern Summary

| Situation | Pattern | Queries |
|---|---|---|
| Load one parent + one related | `INNER JOIN` in single query | 1 |
| Load parent + its children | `LEFT JOIN` + group in Go | 1 |
| Load a list with related data | Batch `IN` query | 2 |
| Optional relations | `With*` options pattern | 1 + N (only requested) |
| Insert and get generated fields | `INSERT ... RETURNING` | 1 |
| Insert many rows | Multi-row `VALUES` | 1 |
| List with pagination | `COUNT` + paginated `SELECT` | 2 |

---

*Next: [12 — Generics & DTO Abstractions →](./12-generics-and-dto-abstractions.md)*
