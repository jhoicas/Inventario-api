package domain

import "errors"

// Errores de dominio (sin dependencias externas).
var (
	ErrNotFound          = errors.New("recurso no encontrado")
	ErrUserNotFound      = errors.New("usuario no encontrado")
	ErrEmailAlreadyExists = errors.New("el email ya está registrado")
	ErrInvalidInput      = errors.New("entrada inválida")
	ErrDuplicate         = errors.New("recurso duplicado")
	ErrUnauthorized      = errors.New("no autorizado")
	ErrForbidden         = errors.New("acceso denegado")
	ErrConflict          = errors.New("conflicto con el estado actual")
	ErrInsufficientStock = errors.New("stock insuficiente")
)
