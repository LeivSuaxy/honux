package domain

// Not Found Error
type ErrNotFound struct {
	Op  string
	Err error
}

func (e *ErrNotFound) Error() string {
	return ""
}

type ErrBadRequest struct {
	Op  string
	Err error
}

func (e *ErrBadRequest) Error() string {
	return ""
}

type ErrValidation struct {
	Op  string
	Err error
}

func (e *ErrValidation) Error() string {
	return ""
}

type ErrConflict struct {
	Op  string
	Err error
}

func (e *ErrConflict) Error() string {
	return ""
}
