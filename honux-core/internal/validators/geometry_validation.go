package validators

import (
	"encoding/json"
	"fmt"
	"honux-core/internal/domain/geometry"
)

func ValidateGeometry(shapeType string, raw json.RawMessage) FieldErrors {
	fe := make(FieldErrors)

	if len(raw) == 0 || string(raw) == "null" {
		fe.Add("geometry", "geometry is required")
		return fe
	}

	var probe any
	if err := json.Unmarshal(raw, &probe); err != nil {
		fe.Add("geometry", "must be a valid JSON")
		return fe
	}

	switch shapeType {
	case geometry.Polygon:
		var g struct {
			Points []struct {
				X *float64 `json:"x"`
				Y *float64 `json:"y"`
			} `json:"points"`
		}
		if err := json.Unmarshal(raw, &g); err != nil {
			fe.Add("geometry", "polygon geometry must have a 'points' array")
			return fe
		}
		if len(g.Points) < 3 {
			fe.Add("geometry", "polygon must have at least 3 points")
			return fe
		}
		for i, p := range g.Points {
			if p.X == nil || p.Y == nil {
				fe.Add("geometry", fmt.Sprintf("point at index %d must have x and y", i))
			}
		}
	case geometry.Rectangle:
		var g struct {
			X      *float64 `json:"x"`
			Y      *float64 `json:"y"`
			Width  *float64 `json:"width"`
			Height *float64 `json:"height"`
		}
		if err := json.Unmarshal(raw, &g); err != nil {
			fe.Add("geometry", "rectangle geometry must have x, y, width, height")
			return fe
		}
		if g.X == nil || g.Y == nil || g.Width == nil || g.Height == nil {
			fe.Add("geometry", "rectangle geometry requires x, y, width, height")
			return fe
		}
		if *g.Width <= 0 || *g.Height <= 0 {
			fe.Add("geometry", "rectangle width and height must be greater than 0")
			return fe
		}
	case geometry.Circle:
		var g struct {
			Center struct {
				X *float64 `json:"x"`
				Y *float64 `json:"y"`
			} `json:"center"`
			Radius *float64 `json:"radius"`
		}
		if err := json.Unmarshal(raw, &g); err != nil {
			fe.Add("geometry", "circle geometry must have center and radius")
			return fe
		}
		if g.Center.X == nil || g.Center.Y == nil {
			fe.Add("geometry", "circle center must have x and y")
			return fe
		}
		if g.Radius == nil || *g.Radius <= 0 {
			fe.Add("geometry", "circle radius must be greater than 0")
			return fe
		}
	default:
		fe.Add("geometry", fmt.Sprintf("unsupported shape_type %q: must be polygon, rectangle or circle", shapeType))
		return fe
	}
	return nil
}

func ValidateShapeType(shapeType string) FieldErrors {
	switch shapeType {
	case geometry.Polygon:
		return nil
	case geometry.Rectangle:
		return nil
	case geometry.Circle:
		return nil
	}

	fe := make(FieldErrors)
	fe.Add("shape_type", "invalid shape_type")
	return fe
}
