package schemas

import (
	"encoding/json"

	"github.com/google/uuid"
)

type CreateUpdateZone struct {
	Name            string          `json:"name"`
	ShortIdentifier *string         `json:"short_identifier,omitempty"`
	ShapeType       string          `json:"shape_type"`
	Geometry        json.RawMessage `json:"geometry"` // TODO Why rawmessage?
	Color           *string         `json:"color,omitempty"`
	FloorId         *uuid.UUID      `json:"floor_id,omitempty"`
}
