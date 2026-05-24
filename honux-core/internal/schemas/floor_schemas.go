package schemas

import (
	"fmt"
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
