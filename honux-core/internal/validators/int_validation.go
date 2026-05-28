package validators

type IntValidator struct {
	FieldErrors FieldErrors `json:"field_errors"`
	field       string
	n           int
}

func NewIntValidator(field string, n int) *IntValidator {
	return &IntValidator{FieldErrors: make(FieldErrors), field: field, n: n}
}

func (sv *IntValidator) IsLessThan(limit int) *IntValidator {
	if sv.n < limit {
		sv.FieldErrors.Add(sv.field, "cannot be less than limit")
	}
	return sv
}

func (sv *IntValidator) IsGreaterThan(limit int) *IntValidator {
	if sv.n > limit {
		sv.FieldErrors.Add(sv.field, "cannot be greater than limit")
	}
	return sv
}

func (sv *IntValidator) IsGreaterThanOrEqualTo(limit int) *IntValidator {
	if sv.n >= limit {
		sv.FieldErrors.Add(sv.field, "cannot be greater than or equal to limit")
	}
	return sv
}

func (sv *IntValidator) IsLessThanOrEqualTo(limit int) *IntValidator {
	if sv.n <= limit {
		sv.FieldErrors.Add(sv.field, "cannot be less than or equal to limit")
	}
	return sv
}

func (sv *IntValidator) CannotBeZero() *IntValidator {
	if sv.n == 0 {
		sv.FieldErrors.Add(sv.field, "cannot be zero")
	}
	return sv
}

func (sv *IntValidator) CannotBeNegative() *IntValidator {
	if sv.n < 0 {
		sv.FieldErrors.Add(sv.field, "cannot be negative")
	}
	return sv
}

func (sv *IntValidator) CannotBePositive() *IntValidator {
	if sv.n > 0 {
		sv.FieldErrors.Add(sv.field, "cannot be positive")
	}
	return sv
}

func (sv *IntValidator) GetErrors() FieldErrors {
	return sv.FieldErrors
}
