package repository

import "github.com/jhoicas/Inventario-api/internal/domain/entity"

// CategoryRepository persiste categorías de producto en crm_category_product_hub.
type CategoryRepository interface {
	Create(category *entity.Category) error
	GetByID(id string) (*entity.Category, error)
	// GetByCompanyAndName detecta duplicados por empresa+nombre (misma semántica que el índice único del hub).
	GetByCompanyAndName(companyID, name string) (*entity.Category, error)
	Update(category *entity.Category) error
	ListByCompany(companyID string, limit, offset int) ([]*entity.Category, int64, error)
	// Deactivate elimina la categoría del hub; crm_products_hub.category_id pasa a NULL (ON DELETE SET NULL).
	Deactivate(companyID, id string) error
}
