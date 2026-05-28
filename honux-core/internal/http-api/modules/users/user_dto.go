package http_users

import (
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

func (r *CreateUpdateUserRequest) Validate() error {
	fe := make(validators.FieldErrors)

	if strings.TrimSpace(r.Username) == "" {
		fe.Add("username", "username is required")
	}

	// Validate Email
	if valid, emailErrors := validators.ValidateEmail(&r.Email); !valid {
		fe.AddErrors("email", emailErrors)
	}

	if len(r.Password) < 8 {
		fe.Add("password", "password must be at least 8 characters")
	}

	return fe.ToAppError()
}

func (r *CreateUpdateUserRequest) ToSchema() *schemas.CreateUpdateUser {
	return &schemas.CreateUpdateUser{
		Username: r.Username,
		Email:    r.Email,
		Password: r.Password,
		IsAdmin:  r.IsAdmin,
	}
}
