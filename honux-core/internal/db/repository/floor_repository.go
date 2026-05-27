package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"honux-core/internal/db/models"
	"honux-core/internal/schemas"

	"github.com/google/uuid"
)

type FloorRepository struct {
	db *sql.DB
}

func NewFloorRepository(db *sql.DB) *FloorRepository {
	return &FloorRepository{db: db}
}

func (r *FloorRepository) List(ctx context.Context, req *schemas.PaginationParams) ([]models.Floor, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM floors WHERE deleted_at IS NULL`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("FloorRepository.List count: %w", err)
	}

	selectQuery := `
		SELECT id, created_at, updated_at, deleted_at, active, name, level
		FROM floors
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, selectQuery, req.PerPage, req.GetOffset())
	if err != nil {
		return nil, 0, fmt.Errorf("FloorRepository.List query: %w", err)
	}
	defer rows.Close()

	var floors []models.Floor
	for rows.Next() {
		var f models.Floor
		var deletedAt sql.NullTime
		if err := rows.Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt, &deletedAt, &f.Active, &f.Active, &f.Level); err != nil {
			return nil, 0, fmt.Errorf("FloorRepository.List scan: %w", err)
		}
		if deletedAt.Valid {
			f.DeletedAt = &deletedAt.Time
		}
		floors = append(floors, f)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("FloorRepository.List rows: %w", err)
	}

	return floors, total, nil
}

func (r *FloorRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Floor, error) {
	query := `
		SELECT id, created_at, updated_at, deleted_at, active, name, level
		FROM floors
		WHERE id = $1 AND deleted_at IS NULL
	`

	var f models.Floor
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&f.ID,
		&f.CreatedAt,
		&f.UpdatedAt,
		&deletedAt,
		&f.Active,
		&f.Name,
		&f.Level,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("FloorRepository.FindById(id=%s): %w", id, err)
	}

	if deletedAt.Valid {
		f.DeletedAt = &deletedAt.Time
	}

	return &f, nil
}

func (r *FloorRepository) Create(ctx context.Context, req *schemas.CreateUpdateFloor) (*models.Floor, error) {
	var f models.Floor
	query := `
		INSERT INTO floors (name, level)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at, active
	`

	err := r.db.QueryRowContext(ctx, query,
		req.Name, req.Level,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt, &f.Active)

	f.Name = req.Name
	f.Level = req.Level

	if err != nil {
		return nil, fmt.Errorf("FloorRepository.Create: %w", err)
	}
	return &f, nil
}

func (r *FloorRepository) Update(ctx context.Context, req *schemas.CreateUpdateFloor, id uuid.UUID) (*models.Floor, error) {
	query := `
		UPDATE floors
		SET
			name  = $1,
			level = $2,
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING id, created_at, updated_at, deleted_at, active, name, level
	`

	var f models.Floor
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query,
		req.Name,
		req.Level,
	).Scan(
		&f.ID, &f.CreatedAt, &f.UpdatedAt, &deletedAt, &f.Active, &f.Name, &f.Level,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("FloorRepository.Update: floor %s not found", id)
		}
		return nil, fmt.Errorf("FloorRepository.Update: %w", err)
	}

	if deletedAt.Valid {
		f.DeletedAt = &deletedAt.Time
	}

	return &f, nil
}

func (r *FloorRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE floors SET deleted_at = NOW(), active = FALSE WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("FloorRepository.Delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("FloorRepository.Delete: floor %s not found", id)
	}
	return nil
}
