package http_zones

import (
	"encoding/json"
	"honux-core/internal/schemas"
	"honux-core/internal/validators"
	"strings"

	"github.com/google/uuid"
)

type CreateUpdateZoneRequest struct {
	Name            string          `json:"name"`
	ShortIdentifier *string         `json:"short_identifier,omitempty"`
	ShapeType       string          `json:"shape_type"`
	Geometry        json.RawMessage `json:"geometry"` // TODO why rawmessage?
	Color           *string         `json:"color,omitempty"`
	FloorId         *string         `json:"floor_id,omitempty"`
}

func (r *CreateUpdateZoneRequest) Validate() error {
	fe := make(validators.FieldErrors)

	// Name validation
	if len(r.Name) > 50 {
		fe.Add("name", "must be less than 50 characters")
	}

	if strings.TrimSpace(r.Name) == "" {
		fe.Add("name", "is required")
	}

	// ShortIdentifier
	if len(*r.ShortIdentifier) > 7 {
		fe.Add("short_identifier", "must be less than 7")
	}

	// ShapeType Validation

	// Geometry Validation

	// Color Validation
	if len(*r.Color) != 7 {
		fe.Add("color", "must be less than 7")
	}

	// Floor ID validation
	if err := uuid.Validate(*r.FloorId); err != nil {
		fe.Add("floor_id", "floor id with invalid uuid format")
	}

	return fe.ToAppError()
}

func (r *CreateUpdateZoneRequest) ToSchema() *schemas.CreateUpdateZone {
	floorId, _ := uuid.Parse(*r.FloorId)

	return &schemas.CreateUpdateZone{
		Name:            r.Name,
		ShortIdentifier: r.ShortIdentifier,
		ShapeType:       r.ShapeType,
		Geometry:        r.Geometry,
		Color:           r.Color,
		FloorId:         &floorId,
	}
}
