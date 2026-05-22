package http_users

import (
	"encoding/json"
	"honux-core/internal/schemas"
	"honux-core/internal/service"
	"honux-core/internal/utils"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type UserHandlerHTTP struct {
	svc *service.UserService
}

func NewUserHandlerHTTP(svc *service.UserService) *UserHandlerHTTP {
	return &UserHandlerHTTP{svc: svc}
}

func (h *UserHandlerHTTP) Create(w http.ResponseWriter, r *http.Request) {
	var req schemas.CreateUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.BadRequest(w, "invalid JSON body") // TODO Missing get errors[]
		return
	}
	defer r.Body.Close()

	user, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		slog.Error("User.Handler.Create", "error", err)
		utils.WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	schemas.Created(w, user)
}

func (h *UserHandlerHTTP) Update(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := uuid.Parse(idStr)

	if err != nil {
		schemas.BadRequest(w, "UUID not valid")
	}

	var req schemas.CreateUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.BadRequest(w, "invalid JSON body") // TODO Missing get errors[]
		return
	}
	defer r.Body.Close()

	user, err := h.svc.Update(r.Context(), &req, id)
	schemas.OK(w, user)
}

func (h *UserHandlerHTTP) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := uuid.Parse(idStr)

	if err != nil {
		schemas.BadRequest(w, "UUID not valid")
	}
	result, err := h.svc.GetByID(r.Context(), id)

	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
	}
	schemas.OK(w, result)
}

func (h *UserHandlerHTTP) List(w http.ResponseWriter, r *http.Request) {
	params, err := schemas.ParsePagination(r)
	if err != nil {
		schemas.BadRequest(w, err.Error())
		return
	}

	users, total, err := h.svc.List(r.Context(), &params)
	if err != nil {
		slog.Error("User.Handler.GetAll", "error", err)
		schemas.InternalError(w)
		return
	}

	schemas.PaginatedOK(w, users, total, params.Page, params.PerPage)
}

func (h *UserHandlerHTTP) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := uuid.Parse(idStr)

	if err != nil {
		schemas.BadRequest(w, "UUID not valid")
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		schemas.InternalError(w)
	}

	schemas.OK(w, "User deleted successfully")
}
