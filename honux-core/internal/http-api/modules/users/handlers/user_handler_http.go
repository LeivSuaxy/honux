package http_users

import (
	"encoding/json"
	"fmt"
	"honux-core/internal/schemas"
	"honux-core/internal/service"
	"log/slog"
	"net/http"
	"strconv"
)

type UserHandlerHTTP struct {
	svc *service.UserService
}

func NewUserHandlerHTTP(svc *service.UserService) *UserHandlerHTTP {
	return &UserHandlerHTTP{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writeJSON encode error", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func pathID(r *http.Request, key string) (int64, error) {
	// Go 1.22+ path value extraction
	s := r.PathValue(key)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %q", key, s)
	}
	return id, nil
}

func (h *UserHandlerHTTP) Create(w http.ResponseWriter, r *http.Request) {
	var req schemas.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	defer r.Body.Close()

	user, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		slog.Error("handler.Create", "error", err)
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, user)
}
