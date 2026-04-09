package billing

import (
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// CustomerUseCase casos de uso para clientes (facturación).
type CustomerUseCase struct {
	repo repository.CustomerRepository
}

type customerListWithFiltersRepository interface {
	ListByCompanyWithFilters(companyID string, filters repository.CustomerListFilters, limit, offset int) ([]*entity.Customer, int64, error)
}

// NewCustomerUseCase construye el caso de uso.
func NewCustomerUseCase(repo repository.CustomerRepository) *CustomerUseCase {
	return &CustomerUseCase{repo: repo}
}

// Create crea un nuevo cliente.
func (uc *CustomerUseCase) Create(companyID string, in dto.CreateCustomerRequest) (*dto.CustomerResponse, error) {
	if in.Name == "" || in.TaxID == "" {
		return nil, domain.ErrInvalidInput
	}
	existing, _ := uc.repo.GetByCompanyAndTaxID(companyID, in.TaxID)
	if existing != nil {
		return nil, domain.ErrDuplicate
	}
	now := time.Now()
	customer := &entity.Customer{
		ID:        uuid.New().String(),
		CompanyID: companyID,
		Name:      in.Name,
		TaxID:     in.TaxID,
		Email:     in.Email,
		Phone:     in.Phone,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.repo.Create(customer); err != nil {
		return nil, err
	}
	return &dto.CustomerResponse{
		ID:           customer.ID,
		CompanyID:    customer.CompanyID,
		Name:         customer.Name,
		TaxID:        customer.TaxID,
		Email:        customer.Email,
		Phone:        customer.Phone,
		LTV:          customer.LTV,
		CategoryName: customer.CategoryName,
	}, nil
}

// Deactivate desactiva un cliente (soft delete).
func (uc *CustomerUseCase) Deactivate(companyID, customerID string) error {
	if companyID == "" || customerID == "" {
		return domain.ErrInvalidInput
	}
	current, err := uc.repo.GetByID(customerID)
	if err != nil {
		return err
	}
	if current == nil {
		return domain.ErrNotFound
	}
	if current.CompanyID != companyID {
		return domain.ErrForbidden
	}
	return uc.repo.SetActive(companyID, customerID, false)
}

// List lista clientes de la empresa.
func (uc *CustomerUseCase) List(companyID string, search string, limit, offset int) (*dto.CustomerListResponse, error) {
	return uc.ListWithFilters(companyID, dto.CustomerListFilters{Search: search}, limit, offset)
}

// ListWithFilters lista clientes de la empresa con filtros avanzados.
func (uc *CustomerUseCase) ListWithFilters(companyID string, filters dto.CustomerListFilters, limit, offset int) (*dto.CustomerListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	repoFilters := repository.CustomerListFilters{
		Search:             filters.Search,
		CategoryID:         firstNonEmpty(filters.CategoryID, filters.CategoryIDFallback),
		CategoryNameLegacy: filters.CategoryName,
		WithoutCategory:    filters.WithoutCategory,
	}
	list, total, err := uc.listByFilters(companyID, repoFilters, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]dto.CustomerResponse, 0, len(list))
	for _, c := range list {
		out = append(out, dto.CustomerResponse{
			ID:           c.ID,
			CompanyID:    c.CompanyID,
			Name:         c.Name,
			TaxID:        c.TaxID,
			Email:        c.Email,
			Phone:        c.Phone,
			LTV:          c.LTV,
			CategoryName: c.CategoryName,
		})
	}
	return &dto.CustomerListResponse{Items: out, Total: int(total), Limit: limit, Offset: offset}, nil
}

func (uc *CustomerUseCase) listByFilters(companyID string, filters repository.CustomerListFilters, limit, offset int) ([]*entity.Customer, int64, error) {
	if advancedRepo, ok := uc.repo.(customerListWithFiltersRepository); ok {
		return advancedRepo.ListByCompanyWithFilters(companyID, filters, limit, offset)
	}
	return uc.repo.ListByCompany(companyID, filters.Search, limit, offset)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// Update actualiza un cliente existente de la empresa.
func (uc *CustomerUseCase) Update(companyID, customerID string, in dto.UpdateCustomerRequest) (*dto.CustomerResponse, error) {
	if customerID == "" || in.Name == "" || in.TaxID == "" {
		return nil, domain.ErrInvalidInput
	}
	current, err := uc.repo.GetByID(customerID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, domain.ErrNotFound
	}
	if current.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	if in.TaxID != current.TaxID {
		existing, err := uc.repo.GetByCompanyAndTaxID(companyID, in.TaxID)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != customerID {
			return nil, domain.ErrDuplicate
		}
	}
	current.Name = in.Name
	current.TaxID = in.TaxID
	current.Email = in.Email
	current.Phone = in.Phone
	current.UpdatedAt = time.Now()
	if err := uc.repo.Update(current); err != nil {
		return nil, err
	}
	return &dto.CustomerResponse{
		ID:           current.ID,
		CompanyID:    current.CompanyID,
		Name:         current.Name,
		TaxID:        current.TaxID,
		Email:        current.Email,
		Phone:        current.Phone,
		LTV:          current.LTV,
		CategoryName: current.CategoryName,
	}, nil
}
