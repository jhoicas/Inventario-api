package repository

import (
	"github.com/shopspring/decimal"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// ProductRepository define el puerto de persistencia para Product (DIP).
type ProductRepository interface {
	Create(product *entity.Product) error
	GetByID(id string) (*entity.Product, error)
	GetByCompanyAndSKU(companyID, sku string) (*entity.Product, error)
	Update(product *entity.Product) error
	UpdateCost(productID string, cost decimal.Decimal) error
	ListByCompany(companyID string, limit, offset int) ([]*entity.Product, error)
	Delete(id string) error
}
