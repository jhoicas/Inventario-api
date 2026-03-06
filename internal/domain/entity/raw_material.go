package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// RawMaterial representa una materia prima utilizada para producir productos terminados.
type RawMaterial struct {
	ID          string
	CompanyID   string
	Name        string
	SKU         string
	Cost        decimal.Decimal
	UnitMeasure string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

