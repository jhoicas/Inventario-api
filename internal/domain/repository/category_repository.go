package repository

import "github.com/tu-usuario/inventory-pro/internal/domain/entity"

// CategoryRepository define el puerto de persistencia para Category (DIP).
type CategoryRepository interface {
	Create(category *entity.Category) error
	GetByID(id string) (*entity.Category, error)
	GetByCompanyAndCode(companyID, code string) (*entity.Category, error)
	Update(category *entity.Category) error
	ListByCompany(companyID string, limit, offset int) ([]*entity.Category, error)
	ListByParent(companyID, parentID string) ([]*entity.Category, error)
	Delete(id string) error
}
