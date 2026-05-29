package service

import (
	"context"
	"honux-core/internal/db/models"
	"honux-core/internal/db/repository"
	"honux-core/internal/schemas"

	"github.com/google/uuid"
)

type ZoneService struct {
	repo *repository.ZoneRepository
}

func NewZoneService(repo *repository.ZoneRepository) *ZoneService {
	return &ZoneService{repo: repo}
}

func (s *ZoneService) List(ctx context.Context, req *schemas.PaginationParams) ([]models.Zone, int, error) {
	return s.repo.List(ctx, req, repository.WithFloor())
}

func (s *ZoneService) GetByID(ctx context.Context, id uuid.UUID) (*models.Zone, error) {
	return s.repo.FindByID(ctx, id, repository.WithFloor())
}

func (s *ZoneService) Create(ctx context.Context, req *schemas.CreateUpdateZone) (*models.Zone, error) {
	return s.repo.Create(ctx, req)
}

func (s *ZoneService) Update(ctx context.Context, req *schemas.CreateUpdateZone, id uuid.UUID) (*models.Zone, error) {
	return s.repo.Update(ctx, req, id)
}

func (s *ZoneService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}
