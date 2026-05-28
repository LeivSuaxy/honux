package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
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

func IsUniqueViolation(err error) bool     { return PgErrCode(err) == pgUniqueViolation }
func IsForeignKeyViolation(err error) bool { return PgErrCode(err) == pgForeignKeyViolation }
func IsNotNullViolation(err error) bool    { return PgErrCode(err) == pgNotNullViolation }
