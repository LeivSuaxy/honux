package http_floors

import (
	"encoding/json"
	"honux-core/internal/schemas"
	"honux-core/internal/service"
	"honux-core/internal/utils"
	"io"
	"log/slog"
	"net/http"
)

type FloorHandlerHTTP struct {
	svc      *service.FloorService
}

func NewFloorHandlerHTTP(svc *service.FloorService) *FloorHandlerHTTP {
	return &FloorHandlerHTTP{svc: svc}
}

func (h *FloorHandlerHTTP) List(w http.ResponseWriter, r *http.Request) {
	params, err := schemas.ParsePagination(r)
	if err != nil {
		schemas.BadRequest(w, err.Error())
		return
	}

	users, total, err := h.svc.List(r.Context(), &params)
	if err != nil {
		slog.Error("Floor.Handler.List", "error", err)
		schemas.InternalError(w)
		return
	}

	schemas.PaginatedOK(w, users, total, params.Page, params.PerPage)
}

func (h *FloorHandlerHTTP) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ExtractPathUUID(r, "/floors/")

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

func (h *FloorHandlerHTTP) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUpdateFloorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.BadRequest(w, "invalid JSON body")
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

	floor, err := h.svc.Create(r.Context(), req.ToSchema())
	if err != nil {
		slog.Error("Floor.Handler.Create", "error", err)
		schemas.RespondError(w, r, err)
		return
	}
	schemas.Created(w, floor)
}

func (h *FloorHandlerHTTP) Update(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ExtractPathUUID(r, "/floors/")

	if err != nil {
		schemas.BadRequest(w, "UUID not valid")
		return
	}

	var req CreateUpdateFloorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.BadRequest(w, "invalid JSON body")
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

	floor, err := h.svc.Update(r.Context(), req.ToSchema(), *id)

	if err != nil {
		schemas.RespondError(w, r, err)
		return
	}

	schemas.OK(w, floor)
}

func (h *FloorHandlerHTTP) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ExtractPathUUID(r, "/floors/")

	if err != nil {
		schemas.BadRequest(w, "UUID not valid")
		return
	}

	if err := h.svc.Delete(r.Context(), *id); err != nil {
		schemas.RespondError(w, r, err)
		return
	}

	schemas.OK(w, "Floor deleted successfully")
}
