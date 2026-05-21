package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AILog struct {
	ID         uuid.UUID `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	Prompt     string    `json:"json"`
	Result     string    `json:"result"`
	Tokens     int       `json:"tokens"`
	Model      *string   `json:"model,omitempty"`
	ExecutedBy *string   `json:"executed_by,omitempty"`
}

type ComponentLog struct {
	ID          uuid.UUID       `json:"id"`
	CreatedAt   time.Time       `json:"created_at"`
	ComponentID uuid.UUID       `json:"component_id"`
	Type        string          `json:"type"`
	Value       float64         `json:"value"`
	Unit        *string         `json:"unit,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}
