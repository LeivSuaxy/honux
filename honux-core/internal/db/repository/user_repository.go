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

func (r *UserRepository) Create(ctx context.Context, req *schemas.CreateUserRequest) (*models.User, error) {
	var u models.User
	query := `
		INSERT INTO users (username, password_hash, email, is_admin)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at, active
	`

	err := r.db.QueryRowContext(ctx, query,
		req.Username, req.Password, req.Email, req.IsAdmin,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.Active)

	if err != nil {
		return nil, fmt.Errorf("UserRepository.Create: %w", err)
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
