package repository

import (
	"context"
	"database/sql"
	"fmt"
	"honux-core/internal/db/models"
	"honux-core/internal/schemas"
)

type ZoneRepository struct {
	db *sql.DB
}

func NewZoneRepository(db *sql.DB) *ZoneRepository {
	return &ZoneRepository{db: db}
}

func (r *ZoneRepository) List(ctx context.Context, req *schemas.PaginationParams) ([]models.Zone, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM zones WHERE deleted_at IS NULL`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("ZoneRepository.List count: %w", err)
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
		return nil, 0, fmt.Errorf("ZoneRepository.List query: %w", err)
	}
	defer rows.Close()

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
		return nil, 0, fmt.Errorf("ZoneRepository.List rows: %w", err)
	}

	return zones, total, nil
}

//func (r *ZoneRepository) Create(ctx context.Context, req *schemas.)
