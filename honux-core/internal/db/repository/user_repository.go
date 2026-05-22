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

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) List(ctx context.Context, req *schemas.PaginationParams) ([]models.User, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("UserRepository.List count: %w", err)
	}

	selectQuery := `
		SELECT id, created_at, updated_at, deleted_at, active, username, email, is_admin
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, selectQuery, req.PerPage, req.GetOffset())
	if err != nil {
		return nil, 0, fmt.Errorf("UserRepository.List query: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		var deletedAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &deletedAt, &u.Active, &u.Username, &u.Email, &u.IsAdmin); err != nil {
			return nil, 0, fmt.Errorf("UserRepository.List scan: %w", err)
		}
		if deletedAt.Valid {
			u.DeletedAt = &deletedAt.Time
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("UserRepository.List rows: %w", err)
	}

	return users, total, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, created_at, updated_at, deleted_at, active, username, password_hash, email, is_admin
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	var u models.User
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.CreatedAt,
		&u.UpdatedAt,
		&deletedAt,
		&u.Active,
		&u.Username,
		&u.PasswordHash,
		&u.Email,
		&u.IsAdmin,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("UserRepository.FindById(id=%s): %w", id, err)
	}

	if deletedAt.Valid {
		u.DeletedAt = &deletedAt.Time
	}

	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, req *schemas.CreateUpdateUserRequest) (*models.User, error) {
	var u models.User
	query := `
		INSERT INTO users (username, password_hash, email, is_admin)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at, active
	`

	err := r.db.QueryRowContext(ctx, query,
		req.Username, req.Password, req.Email, req.IsAdmin,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.Active)

	// Fill missing properties
	u.Username = req.Username
	u.Email = req.Email
	u.IsAdmin = *req.IsAdmin

	if err != nil {
		return nil, fmt.Errorf("UserRepository.Create: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) Update(ctx context.Context, req *schemas.CreateUpdateUserRequest, id uuid.UUID) (*models.User, error) {
	query := `
		UPDATE users
		SET
			username      = $1,
			email         = $2,
			password_hash = $3,
			is_admin      = $4,
			updated_at    = NOW()
		WHERE id = $5 AND deleted_at IS NULL
		RETURNING id, created_at, updated_at, deleted_at, active, username, password_hash, email, is_admin
	`

	var u models.User
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query,
		req.Username,
		req.Email,
		req.Password,
		req.IsAdmin,
		id,
	).Scan(
		&u.ID, &u.CreatedAt, &u.UpdatedAt, &deletedAt,
		&u.Active, &u.Username, &u.PasswordHash, &u.Email, &u.IsAdmin,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("UserRepository.Update: user %s not found", id)
		}
		return nil, fmt.Errorf("UserRepository.Update: %w", err)
	}

	if deletedAt.Valid {
		u.DeletedAt = &deletedAt.Time
	}

	return &u, nil
}

func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET deleted_at = NOW(), active = FALSE WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("UserRepository.Delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("UserRepository.Delete: user %s not found", id)
	}
	return nil
}
