package postgres

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// isUniqueViolation verifica si un error es una violación de constraint único (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" // unique_violation
	}
	return strings.Contains(err.Error(), "23505")
}

// isUndefinedColumn verifica error por columna inexistente (42703).
func isUndefinedColumn(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "42703" // undefined_column
	}
	return strings.Contains(err.Error(), "42703")
}

// isUndefinedTable verifica error por tabla o relación inexistente (42P01).
func isUndefinedTable(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "42P01" // undefined_table
	}
	return strings.Contains(err.Error(), "42P01")
}
