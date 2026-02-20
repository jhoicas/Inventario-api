package entity

import "time"

// Customer representa un cliente de la empresa (facturación).
type Customer struct {
	ID        string
	CompanyID string
	Name      string
	TaxID     string // NIT o Cédula (Colombia)
	Email     string
	Phone     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
