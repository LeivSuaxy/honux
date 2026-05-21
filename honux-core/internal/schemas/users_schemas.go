package schemas

import (
	"fmt"
	"honux-core/internal/schemas/validators"
	"strings"
)

type CreateUserRequest struct {
	Username string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IsAdmin  *bool  `json:"is_admin,omitempty"`
}

func (r *CreateUserRequest) Validate() []error {
	var errors []error

	if strings.TrimSpace(r.Username) == "" {
		errors = append(errors, fmt.Errorf("name is required"))
	}

	// Validate Email
	if valid, emailErrors := validators.ValidateEmail(&r.Email); !valid {
		errors = append(errors, emailErrors...)
	}

	if len(r.Password) < 8 {
		errors = append(errors, fmt.Errorf("password must be at least 8 characters long"))
	}

	if len(errors) == 0 {
		return nil
	}

	return errors
}
