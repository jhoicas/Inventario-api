package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// CrmProductHub representa una fila en crm_products_hub (catálogo paralelo CRM / hub).
type CrmProductHub struct {
	ID          string           `json:"id"`
	CompanyID   string           `json:"company_id"`
	CategoryID  string           `json:"category_id"`
	ProductCode string           `json:"product_code"`
	ProductName string           `json:"product_name"`
	UnitCost    *decimal.Decimal `json:"unit_cost"`
	IsActive    bool             `json:"is_active"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}
