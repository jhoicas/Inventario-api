package usecase

import (
	"time"

	"github.com/google/uuid"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/domain"
	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
)

// CompanyUseCase aplica reglas de negocio para empresas (casos de uso).
type CompanyUseCase struct {
	repo repository.CompanyRepository
}

// NewCompanyUseCase construye el caso de uso con el puerto de persistencia.
func NewCompanyUseCase(repo repository.CompanyRepository) *CompanyUseCase {
	return &CompanyUseCase{repo: repo}
}

// Create crea una nueva empresa. Genera ID y estado inicial. Devuelve domain.ErrDuplicate si el NIT ya existe.
func (uc *CompanyUseCase) Create(in dto.CreateCompanyRequest) (*dto.CompanyResponse, error) {
	existing, _ := uc.repo.GetByNIT(in.NIT)
	if existing != nil {
		return nil, domain.ErrDuplicate
	}
	now := time.Now()
	company := &entity.Company{
		ID:        uuid.New().String(),
		Name:      in.Name,
		NIT:       in.NIT,
		Address:   in.Address,
		Phone:     in.Phone,
		Email:     in.Email,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.repo.Create(company); err != nil {
		return nil, err
	}
	return entityToCompanyResponse(company), nil
}

// GetByID obtiene una empresa por ID.
func (uc *CompanyUseCase) GetByID(id string) (*dto.CompanyResponse, error) {
	company, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, nil
	}
	return entityToCompanyResponse(company), nil
}

// List lista empresas con paginación.
func (uc *CompanyUseCase) List(limit, offset int) (*dto.CompanyListResponse, error) {
	list, err := uc.repo.List(limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]dto.CompanyResponse, 0, len(list))
	for _, c := range list {
		items = append(items, *entityToCompanyResponse(c))
	}
	return &dto.CompanyListResponse{
		Items: items,
		Page:  dto.PageResponse{Limit: limit, Offset: offset},
	}, nil
}

func entityToCompanyResponse(c *entity.Company) *dto.CompanyResponse {
	if c == nil {
		return nil
	}
	return &dto.CompanyResponse{
		ID:        c.ID,
		Name:      c.Name,
		NIT:       c.NIT,
		Address:   c.Address,
		Phone:     c.Phone,
		Email:     c.Email,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
