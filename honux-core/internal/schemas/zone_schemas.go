package schemas

import "encoding/json"

type CreateUpdateZoneRequest struct {
	Name            string          `json:"name"`
	ShortIdentifier *string         `json:"short_identifier,omitempty"`
	ShapeType       string          `json:"shape_type"`
	Geometry        json.RawMessage `json:"geometry"`
	Color           *string         `json:"color,omitempty"`
	FloorId         *string         `json:"floor_id,omitempty"`
}
