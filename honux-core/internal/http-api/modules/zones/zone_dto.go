package http_zones

import (
	"encoding/json"
	"honux-core/internal/schemas"
	"honux-core/internal/validators"

	"github.com/google/uuid"
)

type CreateUpdateZoneRequest struct {
	Name            string          `json:"name"`
	ShortIdentifier *string         `json:"short_identifier,omitempty"`
	ShapeType       string          `json:"shape_type"`
	Geometry        json.RawMessage `json:"geometry"`
	Color           *string         `json:"color,omitempty"`
	FloorId         *string         `json:"floor_id,omitempty"`
}

func (r *CreateUpdateZoneRequest) Validate() error {
	fe := make(validators.FieldErrors)

	nameErrors := validators.NewStringValidator("name", r.Name).
		NotEmpty().
		MinLength(8).
		MaxLength(255).
		GetErrors()

	if nameErrors != nil {
		fe.AppendFieldError(nameErrors)
	}

	// ShortIdentifier
	if r.ShortIdentifier != nil {
		if len(*r.ShortIdentifier) > 7 {
			fe.Add("short_identifier", "must be less than 7")
		}
	}

	// ShapeType Validation
	shapeTypeErrors := validators.ValidateShapeType(r.ShapeType)

	if shapeTypeErrors != nil {
		fe.AppendFieldError(shapeTypeErrors)
	}

	// Geometry Validation
	geometryErrors := validators.ValidateGeometry(r.ShapeType, r.Geometry)

	if geometryErrors != nil {
		fe.AppendFieldError(geometryErrors)
	}

	// Color Validation
	if r.Color != nil {
		if len(*r.Color) != 7 {
			fe.Add("color", "must be less than 7")
		}
	}

	// Floor ID validation
	if r.FloorId != nil {
		if err := uuid.Validate(*r.FloorId); err != nil {
			fe.Add("floor_id", "floor id with invalid uuid format")
		}
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
