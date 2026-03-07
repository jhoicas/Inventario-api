package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// CRMCategory representa una categoría de fidelización (Oro, Plata, Bronce).
type CRMCategory struct {
	ID        string
	CompanyID string
	Name      string          // ej. Oro, Plata, Bronce
	MinLTV    decimal.Decimal // LTV mínimo para pertenecer (opcional)
	CreatedAt time.Time
	UpdatedAt time.Time
}
