package http_users

import (
	"fmt"
	"honux-core/internal/schemas"
	"honux-core/internal/validators"
	"strings"
)

type CreateUpdateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IsAdmin  *bool  `json:"is_admin,omitempty"`
}

func (r *CreateUpdateUserRequest) Validate() []error {
	var errors []error

	if strings.TrimSpace(r.Username) == "" {
		errors = append(errors, fmt.Errorf("username is required"))
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

func (r *CreateUpdateUserRequest) ToSchema() *schemas.CreateUpdateUser {
	return &schemas.CreateUpdateUser{
		Username: r.Username,
		Email:    r.Email,
		Password: r.Password,
		IsAdmin:  r.IsAdmin,
	}
}
