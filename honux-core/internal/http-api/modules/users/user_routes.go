package http_users

import (
	"honux-core/internal/server/router"
)

func RegisterRoutes(r router.Router, h *UserHandlerHTTP) {
	m := r.Module("users")
	m.HandleFunc("GET /users", h.List)
	m.HandleFunc("GET /users/{id}", h.GetByID)
	m.HandleFunc("POST /users", h.Create)
	m.HandleFunc("PUT /users/{id}", h.Update)
	m.HandleFunc("DELETE /users/{id}", h.Delete)
}
