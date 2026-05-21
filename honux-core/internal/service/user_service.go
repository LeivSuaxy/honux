package service

import (
	"context"
	"honux-core/internal/db/models"
	"honux-core/internal/db/repository"
	"honux-core/internal/schemas"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Create(ctx context.Context, req *schemas.CreateUserRequest) (*models.User, error) {
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
