package entity

import (
	"encoding/json"
	"time"
)

// ProductHub represents a product in the analytics hub for fair comparison across sales.
type ProductHub struct {
	ID          string
	CompanyID   string
	ProductCode string
	ProductName string
	Category    *string
	UnitCost    *float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SaleHub represents a sales transaction aggregated for analytics with customer email linkage.
type SaleHub struct {
	ID            string
	CompanyID     string
	CustomerEmail string
	CustomerName  *string
	CustomerCity  *string
	SaleDate      time.Time
	TotalAmount   float64
	CostTotal     *float64
	Profit        *float64
	ItemsSnapshot json.RawMessage
	CreatedAt     time.Time
}

// SaleItemHub represents a line item in a sales hub record.
type SaleItemHub struct {
	ID        string
	SalesID   string
	ProductID string
	Quantity  int
	UnitPrice float64
	LineTotal float64
	CreatedAt time.Time
}

// AIAnalyticsRow represents a denormalized row from the v_crm_ai_analytics view.
type AIAnalyticsRow struct {
	CompanyID      string
	Fecha          time.Time
	ClienteNombre  string
	Ciudad         *string
	Producto       string
	Categoria      *string
	Cantidad       int
	PrecioUnitario float64
	IngresoNeto    float64
	CostoTotal     float64
	Utilidad       float64
	CustomerEmail  string
	SaleID         string
	ItemID         string
}

// CustomerChurnRisk representa un cliente potencialmente inactivo para campañas de retención.
type CustomerChurnRisk struct {
	CompanyID        string
	CustomerEmail    string
	CustomerName     string
	FavoriteProduct  string
	LastPurchaseDate time.Time
	DaysInactive     int
}
