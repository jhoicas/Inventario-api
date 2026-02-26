package repository

import "github.com/jhoicas/Inventario-api/internal/domain/entity"

// WarehouseRepository define el puerto de persistencia para Warehouse (DIP).
type WarehouseRepository interface {
	Create(warehouse *entity.Warehouse) error
	GetByID(id string) (*entity.Warehouse, error)
	Update(warehouse *entity.Warehouse) error
	ListByCompany(companyID string, limit, offset int) ([]*entity.Warehouse, error)
	Delete(id string) error
}
