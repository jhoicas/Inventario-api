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
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.repo.Create(customer); err != nil {
		return nil, err
	}
	return &dto.CustomerResponse{
		ID:        customer.ID,
		CompanyID: customer.CompanyID,
		Name:      customer.Name,
		TaxID:     customer.TaxID,
		Email:     customer.Email,
		Phone:     customer.Phone,
	}, nil
}

// List lista clientes de la empresa.
func (uc *CustomerUseCase) List(companyID string, limit, offset int) ([]*dto.CustomerResponse, error) {
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
	out := make([]*dto.CustomerResponse, 0, len(list))
	for _, c := range list {
		out = append(out, &dto.CustomerResponse{
			ID:        c.ID,
			CompanyID: c.CompanyID,
			Name:      c.Name,
			TaxID:     c.TaxID,
			Email:     c.Email,
			Phone:     c.Phone,
		})
	}
	return out, nil
}
