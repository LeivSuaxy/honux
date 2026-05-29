package http_zones

import "honux-core/internal/server/router"

func RegisterRoutes(r router.Router, h *ZoneHandlerHTTP) {
	m := r.Module("zones")
	m.HandleFunc("GET /zones", h.List)
	m.HandleFunc("GET /zones/{id}", h.GetByID)
	m.HandleFunc("POST /zones", h.Create)
	m.HandleFunc("PUT /zones/{id}", h.Update)
	m.HandleFunc("DELETE /zones/{id}", h.Delete)
}
