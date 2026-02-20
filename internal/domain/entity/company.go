package entity

import "time"

// Company representa una organización/tenant del sistema (multi-tenant, enfoque Colombia).
type Company struct {
	ID        string
	Name      string
	NIT       string    // NIT colombiano (con o sin dígito de verificación)
	Address   string
	Phone     string
	Email     string
	Status    string    // active, suspended, inactive
	CreatedAt time.Time
	UpdatedAt time.Time
}
