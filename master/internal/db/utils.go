package db

import (
	"database/sql"

	"github.com/jackc/pgconn"
	"github.com/pkg/errors"
)

// MatchSentinelError checks if the error belongs to specific families of errors
// and ensures that the returned error has the proper type and text.
func MatchSentinelError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	switch pgErrCode(err) {
	case CodeForeignKeyViolation:
		return ErrNotFound
	case CodeUniqueViolation:
		return ErrDuplicateRecord
	}

	return err
}

// MustHaveAffectedRows checks if bun has affected rows in a table or not.
// Returns ErrNotFound if no rows were affected and returns the provided error otherwise.
func MustHaveAffectedRows(result sql.Result, err error) error {
	if err == nil {
		rowsAffected, affectedErr := result.RowsAffected()
		if affectedErr != nil {
			return affectedErr
		}
		if rowsAffected == 0 {
			return ErrNotFound
		}
	}

	return err
}

func pgErrCode(err error) string {
	if e, ok := err.(*pgconn.PgError); ok {
		return e.Code
	}

	return ""
}
