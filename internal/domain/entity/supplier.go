package entity

import "time"

// Supplier representa un proveedor de productos de la empresa (compras).
type Supplier struct {
	ID              string
	CompanyID       string
	Name            string
	NIT             string
	Email           string
	Phone           string
	PaymentTermDays int
	LeadTimeDays    int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
