package schemas

import (
	"errors"
	"net/http"
	"strconv"
)

type PaginationParams struct {
	Page    int `json:"page"`
	PerPage int `json:"limit"`
}

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

func (p *PaginationParams) GetOffset() int {
	return (p.Page - 1) * p.PerPage
}

func ParsePagination(r *http.Request) (PaginationParams, error) {
	q := r.URL.Query()

	pageStr := q.Get("page")
	perPageStr := q.Get("limit")

	page := DefaultPage
	perPage := DefaultPage

	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return PaginationParams{}, errors.New("invalid param 'page'")
		}
	}

	if perPageStr != "" {
		var err error
		perPage, err = strconv.Atoi(perPageStr)
		if err != nil || perPage < 1 {
			return PaginationParams{}, errors.New("invalid param 'limit'")
		}
		if perPage > MaxPerPage {
			perPage = MaxPerPage
		}
	}

	return PaginationParams{
		Page:    page,
		PerPage: perPage,
	}, nil
}
