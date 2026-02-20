package entity

import "time"

// Warehouse representa una bodega o sucursal donde se almacena inventario (multi-bodega).
type Warehouse struct {
	ID        string
	CompanyID string
	Name      string
	Address   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
