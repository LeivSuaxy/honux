package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"honux-core/internal/db"
	"honux-core/internal/db/models"
	"honux-core/internal/domain/apperror"
	"honux-core/internal/schemas"

	"github.com/google/uuid"
)

type ZoneRepository struct {
	db *sql.DB
}

func NewZoneRepository(db *sql.DB) *ZoneRepository {
	return &ZoneRepository{db: db}
}

func (r *ZoneRepository) List(ctx context.Context, req *schemas.PaginationParams, opts ...ZoneFindOption) ([]models.Zone, int, error) {
	/*var total int
	countQuery := `SELECT COUNT(*) FROM zones WHERE deleted_at IS NULL`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, apperror.Internal(err)
	}

	selectQuery := `
		SELECT id, created_at, updated_at, deleted_at, active, floor_id, name, short_identifier, shape_type, geometry, color
		FROM zones
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, selectQuery, req.PerPage, req.GetOffset())
	if err != nil {
		return nil, 0, apperror.Internal(err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			return
		}
	}(rows)

	var zones []models.Zone
	for rows.Next() {
		var z models.Zone
		var deletedAt sql.NullTime
		if err := rows.Scan(
			&z.ID,
			&z.CreatedAt,
			&z.UpdatedAt,
			&deletedAt,
			&z.Active,
			&z.FloorId,
			&z.Name,
			&z.ShortIdentifier,
			&z.ShapeType,
			&z.Geometry,
			&z.Color,
		); err != nil {
			return nil, 0, fmt.Errorf("ZoneRepository.List scan: %w", err)
		}
		if deletedAt.Valid {
			z.DeletedAt = &deletedAt.Time
		}
		zones = append(zones, z)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(err)
	}

	return zones, total, nil*/

	options := &ZoneFindOptions{}
	for _, o := range opts {
		o(options)
	}

	// Base SELECT
	selectCols := `
		z.id, z.created_at, z.updated_at, z.deleted_at, z.active, z.floor_id,
		z.name, z.short_identifier, z.shape_type, z.geometry, z.color
	`
	joins := ""

	if options.WithFloor {
		selectCols += `,
			f.id, f.created_at, f.updated_at, f.deleted_at, f.active, f.name, f.level`
		joins += " INNER JOIN floors f ON f.id = z.floor_id AND f.deleted_at IS NULL"
	}

	// Count Query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM zones z
		%s
		WHERE z.deleted_at IS NULL
	`, joins)

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, apperror.Internal(err)
	}

	// Select
	selectQuery := fmt.Sprintf(`
		SELECT %s
		FROM zones z
		%s
		WHERE z.deleted_at IS NULL
		ORDER BY z.created_at DESC
		LIMIT $1 OFFSET $2
	`, selectCols, joins)

	rows, err := r.db.QueryContext(ctx, selectQuery, req.PerPage, req.GetOffset())
	if err != nil {
		return nil, 0, apperror.Internal(err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var zones []models.Zone
	for rows.Next() {
		z, err := r.scanZoneRow(rows, options)
		if err != nil {
			return nil, 0, apperror.Internal(err)
		}
		zones = append(zones, *z)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, apperror.Internal(err)
	}

	return zones, total, nil
}

func (r *ZoneRepository) FindByID(ctx context.Context, id uuid.UUID, opts ...ZoneFindOption) (*models.Zone, error) {
	options := &ZoneFindOptions{}
	for _, o := range opts {
		o(options)
	}
	selectCols := `
        z.id, z.created_at, z.updated_at, z.deleted_at, z.active,
        z.floor_id, z.name, z.short_identifier, z.shape_type, z.geometry, z.color`
	joins := ""

	if options.WithFloor {
		selectCols += `,
            f.id, f.created_at, f.updated_at, f.deleted_at, f.active, f.name, f.level`
		joins += " INNER JOIN floors f ON f.id = z.floor_id AND f.deleted_at IS NULL"
	}

	query := fmt.Sprintf(`
        SELECT %s
        FROM zones z
        %s
        WHERE z.id = $1 AND z.deleted_at IS NULL
    `, selectCols, joins)

	row := r.db.QueryRowContext(ctx, query, id)

	z, err := r.scanZoneRow(row, options)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("zone")
		}
		return nil, apperror.Internal(err)
	}
	return z, nil
}

func (r *ZoneRepository) Create(ctx context.Context, req *schemas.CreateUpdateZone) (*models.Zone, error) {
	var z models.Zone

	query := `
		INSERT INTO zones (name, short_identifier, shape_type, geometry, color, floor_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at, active
	`

	err := r.db.QueryRowContext(ctx, query,
		req.Name, req.ShortIdentifier, req.ShapeType, req.Geometry, req.Color, req.FloorId,
	).Scan(&z.ID, &z.CreatedAt, &z.UpdatedAt, &z.Active)

	z.Name = req.Name
	z.ShortIdentifier = req.ShortIdentifier
	z.ShapeType = req.ShapeType
	z.Geometry = req.Geometry
	z.Color = req.Color
	z.FloorId = *req.FloorId

	if err != nil {
		return nil, apperror.Internal(err) // TODO More error control with PgError
	}

	return &z, nil
}

func (r *ZoneRepository) Update(ctx context.Context, req *schemas.CreateUpdateZone, id uuid.UUID) (*models.Zone, error) {
	query := `
		UPDATE zones
		SET
		    name = $1,
		    short_identifier = $2,
		    shape_type = $3,
		    geometry = $4,
		    color = $5,
		    floor_id = $6,
		    updated_at = NOW()
		WHERE id = $7 AND deleted_at IS NULL
		RETURNING id, created_at, updated_at, deleted_at, active
	`

	var z models.Zone
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query,
		req.Name,
		req.ShortIdentifier,
		req.ShapeType,
		req.Geometry,
		req.Color,
		req.FloorId,
	).Scan(
		&z.ID, &z.CreatedAt, &z.UpdatedAt, &z.DeletedAt, &z.Active,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("zone")
		}
		return nil, db.PgIdentifyError(err, db.PgErrorHint{Code: db.ForeignKeyViolation, Message: "floor_id maybe no exists"})
	}

	if deletedAt.Valid {
		z.DeletedAt = &deletedAt.Time
	}

	return &z, nil
}

func (r *ZoneRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE zones SET deleted_at = NOW(), active = FALSE WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return apperror.Internal(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperror.NotFound("zone")
	}
	return nil
}

// ---Query Options Pattern---

type ZoneFindOptions struct {
	WithFloor bool
}

type ZoneFindOption func(*ZoneFindOptions)

func WithFloor() ZoneFindOption { return func(o *ZoneFindOptions) { o.WithFloor = true } }

// -- Helpers --
type rowScanner interface {
	Scan(dest ...any) error
}

func (r *ZoneRepository) scanZoneRow(row rowScanner, opts *ZoneFindOptions) (*models.Zone, error) {
	var (
		z          models.Zone
		zDeletedAt sql.NullTime
		shortID    sql.NullString
		color      sql.NullString
		geometry   []byte
	)

	dest := []any{
		&z.ID, &z.CreatedAt, &z.UpdatedAt, &zDeletedAt, &z.Active,
		&z.FloorId, &z.Name, &shortID, &z.ShapeType, &geometry, &color,
	}

	var (
		f          models.Floor
		fDeletedAt sql.NullTime
	)

	if opts.WithFloor {
		dest = append(dest,
			&f.ID, &f.CreatedAt, &f.UpdatedAt, &fDeletedAt, &f.Active, &f.Name, &f.Level,
		)
	}

	if err := row.Scan(dest...); err != nil {
		return nil, err
	}

	if zDeletedAt.Valid {
		z.DeletedAt = &zDeletedAt.Time
	}
	if shortID.Valid {
		z.ShortIdentifier = &shortID.String
	}
	if color.Valid {
		z.Color = &color.String
	}
	if len(geometry) > 0 {
		z.Geometry = geometry
	}

	if opts.WithFloor {
		if fDeletedAt.Valid {
			f.DeletedAt = &fDeletedAt.Time
		}
		z.Floor = &f
	}

	return &z, nil
}
