package usecase

import (
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// UserUseCase aplica reglas de negocio para usuarios.
type UserUseCase struct {
	repo repository.UserRepository
}

// NewUserUseCase construye el caso de uso con el puerto de persistencia.
func NewUserUseCase(repo repository.UserRepository) *UserUseCase {
	return &UserUseCase{repo: repo}
}

// GetByID obtiene un usuario por ID.
func (uc *UserUseCase) GetByID(id string) (*dto.UserResponse, error) {
	user, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}
	return entityToUserResponse(user), nil
}

// GetByEmailAndCompany obtiene un usuario por email y empresa (para login).
func (uc *UserUseCase) GetByEmailAndCompany(email, companyID string) (*dto.UserResponse, error) {
	user, err := uc.repo.GetByEmailAndCompany(email, companyID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}
	return entityToUserResponse(user), nil
}

func entityToUserResponse(u *entity.User) *dto.UserResponse {
	if u == nil {
		return nil
	}
	return &dto.UserResponse{
		ID:        u.ID,
		CompanyID: u.CompanyID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
