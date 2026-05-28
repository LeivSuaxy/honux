package validators

import (
	"honux-core/internal/domain/apperror"
	"strings"
)

type FieldErrors map[string]string

func (fe FieldErrors) Add(field, message string) {
	fe[field] = message
}

func (fe FieldErrors) AddError(field string, errs error) {
	fe.Add(field, errs.Error())
}

func (fe FieldErrors) AddErrors(field string, errs []error) {
	if len(errs) == 0 {
		return
	}
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		if e != nil {
			msgs = append(msgs, e.Error())
		}
	}
	if len(msgs) > 0 {
		fe[field] = strings.Join(msgs, "; ")
	}
}

func (fe FieldErrors) HasErrors() bool {
	return len(fe) > 0
}

func (fe FieldErrors) AppendFieldError(fields FieldErrors) {
	for field, message := range fields {
		fe.Add(field, message)
	}
}

func (fe FieldErrors) ToAppError() error {
	if !fe.HasErrors() {
		return nil
	}
	return apperror.ValidationError(fe)
}

func AddSubValidator(fe FieldErrors, field string, fn func() (bool, []error)) {
	if valid, errs := fn(); !valid {
		fe.AddErrors(field, errs)
	}
}
