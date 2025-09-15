package repository

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// Domain-level errors I prefer to bubble up from repository implementations.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrConflict      = errors.New("conflict")
)

// MapPgError translates common Postgres error codes to domain errors.
// I only map what I expect to handle explicitly at higher layers; everything else passes through.
func MapPgError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return ErrAlreadyExists
		case pgerrcode.ForeignKeyViolation:
			return ErrConflict
		}
	}
	return err
}
