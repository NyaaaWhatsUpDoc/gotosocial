package bundb

import (
	"github.com/jackc/pgconn"
	"github.com/mattn/go-sqlite3"
	"github.com/superseriousbusiness/gotosocial/internal/db"
)

// processPostgresError processes an error, replacing any postgres specific errors with our own error type
func processPostgresError(err error) db.Error {
	// Attempt to cast as postgres
	pgErr, ok := err.(*pgconn.PgError)
	if !ok {
		return err
	}

	// Handle supplied error code:
	// (https://www.postgresql.org/docs/10/errcodes-appendix.html)
	switch pgErr.Code {
	case "23505" /* unique_violation */ :
		return db.ErrAlreadyExists
	default:
		return err
	}
}

// processSQLiteError processes an error, replacing any sqlite specific errors with our own error type
func processSQLiteError(err error) db.Error {
	// Attempt to cast as sqlite
	sqliteErr, ok := err.(*sqlite3.Error)
	if !ok {
		return err
	}

	// Swap-out sqlite errors for our own.
	if sqliteErr.Code == sqlite3.ErrConstraint &&
		(sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique ||
			sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey) {
		return db.ErrAlreadyExists
	}

	return err
}
