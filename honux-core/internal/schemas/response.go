package schemas

import (
	"honux-core/internal/utils"
	"net/http"
)

type APIResponse[T any] struct {
	Data      T          `json:"data,omitempty"`
	Error     string     `json:"error,omitempty"`
	Paginated *Paginated `json:"paginated,omitempty"`
}

type Paginated struct {
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

func OK[T any](w http.ResponseWriter, data T) {
	utils.WriteJSON(w, http.StatusOK, APIResponse[T]{Data: data})
}

func Created[T any](w http.ResponseWriter, data T) {
	utils.WriteJSON(w, http.StatusCreated, APIResponse[T]{Data: data})
}

func BadRequest(w http.ResponseWriter, msg string) {
	utils.WriteJSON(w, http.StatusBadRequest, APIResponse[any]{Error: msg})
}

func InternalError(w http.ResponseWriter) {
	utils.WriteJSON(w, http.StatusInternalServerError, APIResponse[any]{Error: "internal server error"})
}

func NotFound(w http.ResponseWriter, msg string) {
	utils.WriteJSON(w, http.StatusNotFound, APIResponse[any]{Error: msg})
}

func PaginatedOK[T any](w http.ResponseWriter, data T, total, page, perPage int) {
	utils.WriteJSON(w, http.StatusOK, APIResponse[T]{
		Data: data,
		Paginated: &Paginated{
			Total:   total,
			Page:    page,
			PerPage: perPage,
		},
	})
}
