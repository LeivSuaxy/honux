package http_floors

import (
	"fmt"
	"honux-core/internal/schemas"
	"strings"
)

type CreateUpdateFloorRequest struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

func (r *CreateUpdateFloorRequest) Validate() []error {
	var errors []error

	if strings.TrimSpace(r.Name) == "" {
		errors = append(errors, fmt.Errorf("name is required"))
	}

	if len(r.Name) > 255 {
		errors = append(errors, fmt.Errorf("name is too long"))
	}

	if r.Level <= 0 {
		errors = append(errors, fmt.Errorf("level must be a positive number"))
	}

	if len(errors) == 0 {
		return nil
	}

	return errors
}

func (r *CreateUpdateFloorRequest) ToSchema() *schemas.CreateUpdateFloor {
	return &schemas.CreateUpdateFloor{
		Name:  r.Name,
		Level: r.Level,
	}
}
