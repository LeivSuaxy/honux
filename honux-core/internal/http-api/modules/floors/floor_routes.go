package http_floors

import "net/http"

func RegisterRoutes(mux *http.ServeMux, h *FloorHandlerHTTP) {
	mux.HandleFunc("GET /floors", h.List)
	mux.HandleFunc("GET /floors/{id}", h.GetByID)
	mux.HandleFunc("POST /floors", h.Create)
	mux.HandleFunc("DELETE /floors", h.Delete)
	mux.HandleFunc("PUT /floors", h.Update)
}
