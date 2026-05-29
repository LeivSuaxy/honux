package http_zones

import (
	"encoding/json"
	"honux-core/internal/schemas"
	"honux-core/internal/service"
	"honux-core/internal/utils"
	"io"
	"net/http"
)

type ZoneHandlerHTTP struct {
	svc *service.ZoneService
}

func NewZoneHandlerHTTP(svc *service.ZoneService) *ZoneHandlerHTTP {
	return &ZoneHandlerHTTP{svc: svc}
}

func (h *ZoneHandlerHTTP) List(w http.ResponseWriter, r *http.Request) {
	params, err := schemas.ParsePagination(r)
	if err != nil {
		schemas.BadRequest(w, err.Error())
		return
	}

	zones, total, err := h.svc.List(r.Context(), &params)
	if err != nil {
		schemas.RespondError(w, r, err)
		return
	}

	schemas.PaginatedOK(w, zones, total, params.Page, params.PerPage)
}

func (h *ZoneHandlerHTTP) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUpdateZoneRequest
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

	zone, err := h.svc.Create(r.Context(), req.ToSchema())
	if err != nil {
		schemas.RespondError(w, r, err)
		return
	}
	schemas.Created(w, zone)
}

func (h *ZoneHandlerHTTP) Update(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ExtractPathUUID(r, "/zones/")

	if err != nil {
		schemas.BadRequest(w, "UUID not valid")
		return
	}

	var req CreateUpdateZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schemas.RespondError(w, r, err)
		return
	}
	defer func (Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			schemas.RespondError(w, r, err)
		}
	}(r.Body)

	if errs := req.Validate(); errs != nil {
		schemas.RespondError(w, r, errs)
		return
	}

	zone, err := h.svc.Update(r.Context(), req.ToSchema(), *id)

	if err != nil {
		schemas.RespondError(w, r, err)
		return
	}

	schemas.OK(w, zone)
}

func (h *ZoneHandlerHTTP) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ExtractPathUUID(r, "/zones/")

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

func (h *ZoneHandlerHTTP) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ExtractPathUUID(r, "/zones/")

	if err != nil {
		schemas.BadRequest(w, "UUID not valid")
		return
	}

	if err := h.svc.Delete(r.Context(), *id); err != nil {
		schemas.RespondError(w, r, err)
		return
	}

	schemas.OK(w, "Zone deleted successfully")
}