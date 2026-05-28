package http_users

import (
	"honux-core/internal/schemas"
	"honux-core/internal/validators"
)

type CreateUpdateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IsAdmin  *bool  `json:"is_admin,omitempty"`
}

func (r *CreateUpdateUserRequest) Validate() error {
	fe := make(validators.FieldErrors)

	usernameErrors := validators.NewStringValidator("username", r.Username).
		IsNotEmpty().
		IsGreaterThan(255).
		IsLessThan(0).
		GetErrors()

	if usernameErrors != nil {
		fe.AppendFieldError(usernameErrors)
	}

	// Validate Email
	if valid, emailErrors := validators.ValidateEmail(&r.Email); !valid {
		fe.AddErrors("email", emailErrors)
	}

	passwordErrors := validators.NewStringValidator("password", r.Password).
		IsNotEmpty().
		IsLessThan(8).
		GetErrors()

	if passwordErrors != nil {
		fe.AppendFieldError(passwordErrors)
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
