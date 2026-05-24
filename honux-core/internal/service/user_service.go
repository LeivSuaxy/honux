package service

import (
	"context"
	"fmt"
	"honux-core/internal/db/models"
	"honux-core/internal/db/repository"
	"honux-core/internal/schemas"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) List(ctx context.Context, req *schemas.PaginationParams) ([]models.User, int, error) {
	return s.repo.List(ctx, req)
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, fmt.Errorf("user %d not found", id)
	}
	return u, nil
}

// TODO Review returns errors!
func (s *UserService) Create(ctx context.Context, req *schemas.CreateUpdateUserRequest) (*models.User, error) {
	if errors := req.Validate(); errors != nil {
		return nil, nil
	}

	hashed_password, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil
	}

	req.Password = string(hashed_password)

	return s.repo.Create(ctx, req)
}

// TODO Review returns errors!
func (s *UserService) Update(ctx context.Context, req *schemas.CreateUpdateUserRequest, id uuid.UUID) (*models.User, error) {
	if errors := req.Validate(); errors != nil {
		return nil, nil // TODO Capture and emit []errors
	}

	hashed_password, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil
	}

	req.Password = string(hashed_password)

	return s.repo.Update(ctx, req, id)
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.SoftDelete(ctx, id)

	if err != nil {
		return err
	}

	return nil
}
