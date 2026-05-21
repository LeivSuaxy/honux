package models

import "github.com/google/uuid"

type User struct {
	Base
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Email        string `json:"email"`
	IsAdmin      bool   `json:"is_admin"`
}

type UserZonePermission struct {
	Base
	UserId      uuid.UUID   `json:"user_id"`
	ZoneId      uuid.UUID   `json:"zone_id"`
	AccessLevel AccessLevel `json:"access_level"`
	User        *User       `json:"user,omitempty"`
	Zone        *Zone       `json:"zone,omitempty"`
}
