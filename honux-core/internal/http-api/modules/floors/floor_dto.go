package http_floors

import (
	"honux-core/internal/schemas"
	"honux-core/internal/validators"
	"strings"
)

type CreateUpdateFloorRequest struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

func (r *CreateUpdateFloorRequest) Validate() error {
	fe := make(validators.FieldErrors)

	if strings.TrimSpace(r.Name) == "" {
		fe.Add("name", "name is required")
	}

	if len(r.Name) > 255 {
		fe.Add("name", "name is too long")
	}

	if r.Level <= 0 {
		fe.Add("level", "must be greater than zero")
	}

	return fe.ToAppError()
}

func (r *CreateUpdateFloorRequest) ToSchema() *schemas.CreateUpdateFloor {
	return &schemas.CreateUpdateFloor{
		Name:  r.Name,
		Level: r.Level,
	}
}
