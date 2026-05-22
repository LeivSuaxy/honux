package http_users

import (
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, h *UserHandlerHTTP) {
	mux.HandleFunc("GET /users", h.List)
	mux.HandleFunc("GET /users/{id}", h.GetByID)
	mux.HandleFunc("POST /users", h.Create)
	mux.HandleFunc("DELETE /users/{id}", h.Delete)
	mux.HandleFunc("PUT /users/{id}", h.Update)
}
