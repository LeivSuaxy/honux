package schemas

type CreateUpdateUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IsAdmin  *bool  `json:"is_admin,omitempty"`
}
