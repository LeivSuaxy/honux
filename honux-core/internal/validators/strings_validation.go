package validators

import (
	"strings"
)

type StringValidator struct {
	FieldErrors FieldErrors `json:"field_errors"`
	field       string
	s           string
}

func NewStringValidator(field, s string) *StringValidator {
	return &StringValidator{FieldErrors: make(FieldErrors), field: field, s: s}
}

func (sv *StringValidator) NotEmpty() *StringValidator {
	if strings.TrimSpace(sv.s) == "" {
		sv.FieldErrors.Add(sv.field, "cannot be empty")
	}
	return sv
}

func (sv *StringValidator) MaxLength(limit int) *StringValidator {
	if len(sv.s) > limit {
		sv.FieldErrors.Add(sv.field, "cannot be greater than limit")
	}
	return sv
}

func (sv *StringValidator) MinLength(limit int) *StringValidator {
	if len(sv.s) < limit {
		sv.FieldErrors.Add(sv.field, "cannot be less than limit")
	}
	return sv
}

func (sv *StringValidator) GetErrors() FieldErrors {
	return sv.FieldErrors
}
