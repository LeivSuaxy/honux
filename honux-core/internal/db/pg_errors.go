package db

import (
	"errors"
	"honux-core/internal/domain/apperror"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	UniqueViolation     = "23505"
	ForeignKeyViolation = "23503"
	NotNullViolation    = "23502"
	CheckViolation      = "23514"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgNotNullViolation    = "23502"
	pgCheckViolation      = "23514"
)

func PgErrCode(err error) string {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code
	}
	return ""
}

type PgErrorHint struct {
	Code    string
	Message string
}

func hintFor(code string, hints []PgErrorHint) string {
	for _, h := range hints {
		if h.Code == code {
			return h.Message
		}
	}
	return ""
}

func IsUniqueViolation(err error) bool     { return PgErrCode(err) == pgUniqueViolation }
func IsForeignKeyViolation(err error) bool { return PgErrCode(err) == pgForeignKeyViolation }
func IsNotNullViolation(err error) bool    { return PgErrCode(err) == pgNotNullViolation }
func IsCheckViolation(err error) bool      { return PgErrCode(err) == pgCheckViolation }

func PgIdentifyError(err error, hints ...PgErrorHint) error {
	if err == nil {
		return nil
	}

	code := PgErrCode(err)
	switch code {
	case pgUniqueViolation:
		msg := "resource already exists"
		if h := hintFor(pgUniqueViolation, hints); h != "" {
			msg = h
		}
		return apperror.Conflict(msg, err)
	case pgForeignKeyViolation:
		msg := "related resource not found"
		if h := hintFor(pgForeignKeyViolation, hints); h != "" {
			msg = h
		}
		return apperror.NotFound(msg, err)
	case pgNotNullViolation:
		return apperror.BadRequest("a required field is missing", err)
	case pgCheckViolation:
		return apperror.BadRequest("a field value is invalid", err)
	}

	return apperror.Internal(err)
}
