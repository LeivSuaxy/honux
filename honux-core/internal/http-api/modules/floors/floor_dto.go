package http_floors

import (
	"honux-core/internal/schemas"
	"honux-core/internal/validators"
)

type CreateUpdateFloorRequest struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

func (r *CreateUpdateFloorRequest) Validate() error {
	fe := make(validators.FieldErrors)

	nameErrors := validators.
		NewStringValidator("name", r.Name).
		NotEmpty().
		MaxLength(255).
		GetErrors()

	if nameErrors != nil {
		fe.AppendFieldError(nameErrors)
	}

	levelErrors := validators.
		NewIntValidator("level", r.Level).
		CannotBeZero().
		CannotBeNegative().
		GetErrors()

	if levelErrors != nil {
		fe.AppendFieldError(levelErrors)
	}

	return fe.ToAppError()
}

func (r *CreateUpdateFloorRequest) ToSchema() *schemas.CreateUpdateFloor {
	return &schemas.CreateUpdateFloor{
		Name:  r.Name,
		Level: r.Level,
	}
}
