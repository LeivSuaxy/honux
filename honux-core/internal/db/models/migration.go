package models

import (
	"time"

	"github.com/google/uuid"
)

type Migration struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	AppliedAt time.Time `json:"applied_at"`
}
