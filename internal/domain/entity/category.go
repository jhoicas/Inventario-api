package entity

import "time"

// Category representa una categoría de productos (jerárquica opcional).
type Category struct {
	ID        string
	CompanyID string
	ParentID  string    // vacío si es raíz
	Name      string
	Code      string    // código único por empresa
	Status    string    // active, inactive
	CreatedAt time.Time
	UpdatedAt time.Time
}
