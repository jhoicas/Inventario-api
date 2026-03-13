package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

// AdjustmentReasons lista de razones válidas para movimientos de tipo ADJUSTMENT.
var AdjustmentReasons = []string{"MERMA", "ROBO", "VENCIMIENTO", "CONTEO_FISICO", "DETERIORO", "OTRO"}

// RegisterMovementRequest body para POST /api/inventory/movements.
type RegisterMovementRequest struct {
	ProductID       string           `json:"product_id"`
	WarehouseID     string           `json:"warehouse_id,omitempty"`
	FromWarehouseID string           `json:"from_warehouse_id,omitempty"`
	ToWarehouseID   string           `json:"to_warehouse_id,omitempty"`
	Type            string           `json:"type"`
	Quantity        decimal.Decimal  `json:"quantity"`
	UnitCost        *decimal.Decimal `json:"unit_cost,omitempty"`
	// AdjustmentReason es obligatorio cuando Type == "ADJUSTMENT".
	// Valores válidos: MERMA | ROBO | VENCIMIENTO | CONTEO_FISICO | DETERIORO | OTRO
	AdjustmentReason string `json:"adjustment_reason,omitempty"`
}

// ReplenishmentSuggestionDTO representa una sugerencia de reposición para un SKU
// que se encuentra por debajo de su punto de reorden.
type ReplenishmentSuggestionDTO struct {
	ProductID           string          `json:"product_id"`
	SKU                 string          `json:"sku"`
	ProductName         string          `json:"product_name"`
	CurrentStock        decimal.Decimal `json:"current_stock"`
	ReorderPoint        decimal.Decimal `json:"reorder_point"`
	IdealStock          decimal.Decimal `json:"ideal_stock"`          // ReorderPoint * 1.5
	SuggestedOrderQty   decimal.Decimal `json:"suggested_order_qty"`  // IdealStock - CurrentStock
	UnitCost            decimal.Decimal `json:"unit_cost"`            // costo promedio ponderado
	EstimatedOrderCost  decimal.Decimal `json:"estimated_order_cost"` // SuggestedOrderQty * UnitCost
	GrossMarginPct      decimal.Decimal `json:"gross_margin_pct"`     // % margen histórico (puede ser 0 si sin ventas)
	UnitsSoldLast90Days decimal.Decimal `json:"units_sold_last_90d"`  // volumen de ventas reciente
	InventoryDays       decimal.Decimal `json:"inventory_days"`       // días de inventario = CurrentStock / (UnitsSoldLast90Days/90)
	Priority            int             `json:"priority"`             // 1 = más urgente
}

// StockSummaryDTO resumen de stock para un producto (una bodega o agregado de todas).
type StockSummaryDTO struct {
	ProductID      string          `json:"product_id"`
	WarehouseID    string          `json:"warehouse_id,omitempty"` // vacío si es agregado de todas las bodegas
	CurrentStock   decimal.Decimal `json:"current_stock"`
	ReservedStock  decimal.Decimal `json:"reserved_stock"`
	AvailableStock decimal.Decimal `json:"available_stock"`
	AvgCost        decimal.Decimal `json:"avg_cost"`
	LastUpdated    time.Time       `json:"last_updated"`
}

// MovementFiltersDTO filtros para listado de movimientos.
type MovementFiltersDTO struct {
	ProductID   string
	WarehouseID string
	Type        string
	StartDate   time.Time
	EndDate     time.Time
	Limit       int
	Offset      int
}

// MovementDTO movimiento de inventario en respuestas de listado.
type MovementDTO struct {
	ID            string          `json:"id"`
	TransactionID string          `json:"transaction_id"`
	ProductID     string          `json:"product_id"`
	WarehouseID   string          `json:"warehouse_id"`
	Type          string          `json:"type"`
	Quantity      decimal.Decimal `json:"quantity"`
	Balance       decimal.Decimal `json:"balance"`
	UnitCost      decimal.Decimal `json:"unit_cost"`
	TotalCost     decimal.Decimal `json:"total_cost"`
	Notes         string          `json:"notes,omitempty"`
	Date          time.Time       `json:"date"`
	CreatedAt     time.Time       `json:"created_at"`
	CreatedBy     string          `json:"created_by,omitempty"`
}

// PaginatedMovementsDTO respuesta paginada de movimientos.
type PaginatedMovementsDTO struct {
	Items []MovementDTO `json:"items"`
	Total int64         `json:"total"`
}

// Compatibilidad retroactiva con nombres anteriores.
type InventoryMovementFilter = MovementFiltersDTO
type InventoryMovementDTO = MovementDTO
type InventoryMovementListResponse = PaginatedMovementsDTO
