package service

import (
	"context"
	"fmt"
	"honux-core/internal/db/models"
	"honux-core/internal/db/repository"
	"honux-core/internal/schemas"

	"github.com/google/uuid"
)

type FloorService struct {
	repo *repository.FloorRepository
}

func NewFloorService(repo *repository.FloorRepository) *FloorService {
	return &FloorService{repo: repo}
}

func (s *FloorService) List(ctx context.Context, req *schemas.PaginationParams) ([]models.Floor, int, error) {
	return s.repo.List(ctx, req)
}

func (s *FloorService) GetByID(ctx context.Context, id uuid.UUID) (*models.Floor, error) {
	f, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("floor %d not found", id)
	}
	return f, nil
}

func (s *FloorService) Create(ctx context.Context, req *schemas.CreateUpdateFloorRequest) (*models.Floor, error) {
	if errors := req.Validate(); errors != nil {
		return nil, nil
	}

	return s.repo.Create(ctx, req)
}

func (s *FloorService) Update(ctx context.Context, req *schemas.CreateUpdateFloorRequest, id uuid.UUID) (*models.Floor, error) {
	if errors := req.Validate(); errors != nil {
		return nil, nil // TODO Capture and emit []errors
	}

	return s.repo.Update(ctx, req, id)
}

func (s *FloorService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.SoftDelete(ctx, id)

	if err != nil {
		return err
	}

	return nil
}
