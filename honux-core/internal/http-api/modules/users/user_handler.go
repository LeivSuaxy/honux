package http_users

import (
	"encoding/json"
	"honux-core/internal/schemas"
	"honux-core/internal/service"
	"honux-core/internal/utils"
	"io"
	"log/slog"
	"net/http"
)

type UserHandlerHTTP struct {
	svc *service.UserService
}

func NewUserHandlerHTTP(svc *service.UserService) *UserHandlerHTTP {
	return &UserHandlerHTTP{svc: svc}
}

func (h *UserHandlerHTTP) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.RespondError(w, r, err)
		return
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			schemas.RespondError(w, r, err)
			return
		}
	}(r.Body)

	if errs := req.Validate(); errs != nil {
		schemas.RespondError(w, r, errs)
		return
	}

	user, err := h.svc.Create(r.Context(), req.ToSchema())
	if err != nil {
		slog.Error("User.Handler.Create", "error", err)
		schemas.RespondError(w, r, err)
		return
	}
	schemas.Created(w, user)
}

func (h *UserHandlerHTTP) Update(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ExtractPathUUID(r, "/users/")

	if err != nil {
		schemas.BadRequest(w, "UUID not valid")
		return
	}

	var req CreateUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.RespondError(w, r, err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			schemas.RespondError(w, r, err)
		}
	}(r.Body)

	if errs := req.Validate(); errs != nil {
		schemas.RespondError(w, r, err)
		return
	}

	user, err := h.svc.Update(r.Context(), req.ToSchema(), *id)

	if err != nil {
		slog.Error("UserHandler.Update", "error", err)
		schemas.RespondError(w, r, err)
		return
	}

	schemas.OK(w, user)
}

func (h *UserHandlerHTTP) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ExtractPathUUID(r, "/users/")

	if err != nil {
		schemas.BadRequest(w, "UUID not valid")
		return
	}
	result, err := h.svc.GetByID(r.Context(), *id)

	if err != nil {
		schemas.RespondError(w, r, err)
		return
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
		slog.Error("User.Handler.List", "error", err)
		schemas.RespondError(w, r, err)
		return
	}

	schemas.PaginatedOK(w, users, total, params.Page, params.PerPage)
}

func (h *UserHandlerHTTP) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ExtractPathUUID(r, "/users/")

	if err != nil {
		schemas.BadRequest(w, "UUID not valid")
		return
	}

	if err := h.svc.Delete(r.Context(), *id); err != nil {
		schemas.InternalError(w)
		return
	}

	schemas.OK(w, "User deleted successfully")
}
