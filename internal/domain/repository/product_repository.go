package repository

import (
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/shopspring/decimal"
)

// ProductRepository define el puerto de persistencia para Product (DIP).
type ProductRepository interface {
	Create(product *entity.Product) error
	GetByID(id string) (*entity.Product, error)
	GetByCompanyAndSKU(companyID, sku string) (*entity.Product, error)
	Update(product *entity.Product) error
	UpdateCost(productID string, cost decimal.Decimal) error
	ListByCompany(companyID, search string, limit, offset int, activeOnly bool) ([]*entity.Product, int64, error)
	Delete(id string) error
	Deactivate(companyID, id string) error
}
