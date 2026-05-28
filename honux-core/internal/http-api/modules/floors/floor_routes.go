package http_floors

import (
	"honux-core/internal/server/router"
)

func RegisterRoutes(r router.Router, h *FloorHandlerHTTP) {
	m := r.Module("floors")
	m.HandleFunc("GET /floors", h.List)
	m.HandleFunc("GET /floors/{id}", h.GetByID)
	m.HandleFunc("POST /floors", h.Create)
	m.HandleFunc("PUT /floors/{id}", h.Update)
	m.HandleFunc("DELETE /floors/{id}", h.Delete)
}
