package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// CRMCustomerProfile extiende al cliente con datos de fidelización (categoría y LTV).
type CRMCustomerProfile struct {
	ID         string
	CustomerID string
	CompanyID  string
	CategoryID string          // nullable si no asignado
	LTV        decimal.Decimal // lifetime value
	CreatedAt  time.Time
	UpdatedAt time.Time
}

// Profile360 agrupa datos del cliente y su perfil CRM para vista 360 (JOIN customers + crm_customer_profiles).
type Profile360 struct {
	Customer   Customer
	ProfileID  string
	CategoryID string
	LTV        decimal.Decimal
}
