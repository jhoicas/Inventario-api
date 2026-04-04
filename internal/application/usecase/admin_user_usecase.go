package usecase

import (
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"golang.org/x/crypto/bcrypt"
)

// AdminUserUseCase aplica reglas de negocio para usuarios administrados por super_admin.
type AdminUserUseCase struct {
	repo repository.UserRepository
}

// NewAdminUserUseCase construye el caso de uso.
func NewAdminUserUseCase(repo repository.UserRepository) *AdminUserUseCase {
	return &AdminUserUseCase{repo: repo}
}

// ListByCompany lista usuarios de una empresa.
func (uc *AdminUserUseCase) ListByCompany(companyID string, limit, offset int) ([]*dto.UserResponse, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	list, err := uc.repo.ListByCompany(companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]*dto.UserResponse, 0, len(list))
	for _, u := range list {
		out = append(out, entityToUserResponse(u))
	}
	return out, nil
}

// CreateForCompany crea un usuario admin para una empresa.
func (uc *AdminUserUseCase) CreateForCompany(companyID string, in dto.AdminCreateUserRequest) (*dto.UserResponse, error) {
	if companyID == "" || in.Name == "" || in.Email == "" || in.Password == "" || in.Status == "" {
		return nil, domain.ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	user := &entity.User{
		ID:           uuid.New().String(),
		CompanyID:    companyID,
		Email:        in.Email,
		PasswordHash: string(hash),
		Name:         in.Name,
		Roles:        []string{entity.RoleAdmin},
		Status:       in.Status,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := uc.repo.Create(user); err != nil {
		return nil, err
	}
	return entityToUserResponse(user), nil
}

// UpdateForCompany actualiza un usuario de la empresa.
func (uc *AdminUserUseCase) UpdateForCompany(companyID, userID string, in dto.AdminUpdateUserRequest) (*dto.UserResponse, error) {
	if companyID == "" || userID == "" {
		return nil, domain.ErrInvalidInput
	}
	current, err := uc.repo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, domain.ErrNotFound
	}
	if current.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	if in.Name != nil {
		current.Name = *in.Name
	}
	if in.Email != nil {
		current.Email = *in.Email
	}
	if in.Status != nil {
		current.Status = *in.Status
	}
	if in.Password != nil && *in.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*in.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		current.PasswordHash = string(hash)
	}
	current.Roles = []string{entity.RoleAdmin}
	current.UpdatedAt = time.Now()
	if err := uc.repo.Update(current); err != nil {
		return nil, err
	}
	return entityToUserResponse(current), nil
}
