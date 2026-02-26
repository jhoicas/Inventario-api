package repository

import "github.com/jhoicas/Inventario-api/internal/domain/entity"

// UserRepository define el puerto de persistencia para User (DIP).
type UserRepository interface {
	Create(user *entity.User) error
	GetByID(id string) (*entity.User, error)
	GetByEmail(email string) (*entity.User, error)
	GetByEmailAndCompany(email, companyID string) (*entity.User, error)
	Update(user *entity.User) error
	ListByCompany(companyID string, limit, offset int) ([]*entity.User, error)
	Delete(id string) error
	// FindByID y FindByEmail alias semánticos para uso en auth.
	FindByID(id string) (*entity.User, error)
	FindByEmail(email string) (*entity.User, error)
}
