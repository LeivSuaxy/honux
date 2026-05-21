package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Floor struct {
	Base
	Name  string `json:"name"`
	Level int    `json:"level"`
}

type Zone struct {
	Base
	FloorId         uuid.UUID       `json:"floor_id"`
	Name            string          `json:"name"`
	ShortIdentifier *string         `json:"short_identifier,omitempty"`
	ShapeType       string          `json:"shape_type"`
	Geometry        json.RawMessage `json:"geometry,omitempty"`
	Color           *string         `json:"color,omitempty"`
	Floor           *Floor          `json:"floor,omitempty"` // Relation
}
