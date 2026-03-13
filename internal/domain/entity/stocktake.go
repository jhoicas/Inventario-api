package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// Stocktake represents a physical inventory count session
// Status: OPEN | CLOSED
// StocktakeItem: ProductID, SystemQty, CountedQty, Difference decimal.Decimal
type Stocktake struct {
	ID          string
	CompanyID   string
	WarehouseID string
	Status      string
	CreatedAt   time.Time
	ClosedAt    *time.Time
	Items       []StocktakeItem
}

type StocktakeItem struct {
	ID          string
	StocktakeID string
	ProductID   string
	SystemQty   decimal.Decimal
	CountedQty  decimal.Decimal
	Difference  decimal.Decimal
}

const (
	StocktakeStatusOpen   = "OPEN"
	StocktakeStatusClosed = "CLOSED"
)
