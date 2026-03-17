package usecase

import (
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// SupplierUseCase casos de uso CRUD para proveedores.
type SupplierUseCase struct {
	repo repository.SupplierRepository
}

// NewSupplierUseCase construye el caso de uso.
func NewSupplierUseCase(repo repository.SupplierRepository) *SupplierUseCase {
	return &SupplierUseCase{repo: repo}
}

// Create crea un nuevo proveedor.
func (uc *SupplierUseCase) Create(companyID string, in dto.CreateSupplierRequest) (*dto.SupplierResponse, error) {
	if in.Name == "" || in.NIT == "" {
		return nil, domain.ErrInvalidInput
	}
	if in.PaymentTermDays < 0 || in.LeadTimeDays < 0 {
		return nil, domain.ErrInvalidInput
	}

	existing, _ := uc.repo.GetByCompanyAndNIT(companyID, in.NIT)
	if existing != nil {
		return nil, domain.ErrDuplicate
	}

	now := time.Now()
	supplier := &entity.Supplier{
		ID:              uuid.New().String(),
		CompanyID:       companyID,
		Name:            in.Name,
		NIT:             in.NIT,
		Email:           in.Email,
		Phone:           in.Phone,
		PaymentTermDays: in.PaymentTermDays,
		LeadTimeDays:    in.LeadTimeDays,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := uc.repo.Create(supplier); err != nil {
		return nil, err
	}

	return toSupplierResponse(supplier), nil
}

// Deactivate desactiva un proveedor (soft delete).
func (uc *SupplierUseCase) Deactivate(companyID, supplierID string) error {
	if companyID == "" || supplierID == "" {
		return domain.ErrInvalidInput
	}
	current, err := uc.repo.GetByID(supplierID)
	if err != nil {
		return err
	}
	if current == nil {
		return domain.ErrNotFound
	}
	if current.CompanyID != companyID {
		return domain.ErrForbidden
	}
	return uc.repo.SetActive(companyID, supplierID, false)
}

// GetByID obtiene un proveedor por ID.
func (uc *SupplierUseCase) GetByID(id string) (*dto.SupplierResponse, error) {
	supplier, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if supplier == nil {
		return nil, nil
	}
	return toSupplierResponse(supplier), nil
}

// List lista proveedores por empresa con filtros.
func (uc *SupplierUseCase) List(companyID string, filters dto.SupplierFilters) (*dto.SupplierListResponse, error) {
	if filters.Limit <= 0 {
		filters.Limit = 20
	}
	if filters.Limit > 100 {
		filters.Limit = 100
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	list, err := uc.repo.ListByCompany(companyID, filters.Search, filters.Limit, filters.Offset)
	if err != nil {
		return nil, err
	}

	items := make([]dto.SupplierResponse, 0, len(list))
	for _, s := range list {
		items = append(items, *toSupplierResponse(s))
	}

	return &dto.SupplierListResponse{
		Items: items,
		Page:  dto.PageResponse{Limit: filters.Limit, Offset: filters.Offset},
	}, nil
}

// Update actualiza un proveedor existente.
func (uc *SupplierUseCase) Update(id string, in dto.UpdateSupplierRequest) (*dto.SupplierResponse, error) {
	supplier, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if supplier == nil {
		return nil, nil
	}

	if in.Name != nil {
		if *in.Name == "" {
			return nil, domain.ErrInvalidInput
		}
		supplier.Name = *in.Name
	}
	if in.NIT != nil {
		if *in.NIT == "" {
			return nil, domain.ErrInvalidInput
		}
		if *in.NIT != supplier.NIT {
			existing, _ := uc.repo.GetByCompanyAndNIT(supplier.CompanyID, *in.NIT)
			if existing != nil && existing.ID != supplier.ID {
				return nil, domain.ErrDuplicate
			}
		}
		supplier.NIT = *in.NIT
	}
	if in.Email != nil {
		supplier.Email = *in.Email
	}
	if in.Phone != nil {
		supplier.Phone = *in.Phone
	}
	if in.PaymentTermDays != nil {
		if *in.PaymentTermDays < 0 {
			return nil, domain.ErrInvalidInput
		}
		supplier.PaymentTermDays = *in.PaymentTermDays
	}
	if in.LeadTimeDays != nil {
		if *in.LeadTimeDays < 0 {
			return nil, domain.ErrInvalidInput
		}
		supplier.LeadTimeDays = *in.LeadTimeDays
	}

	supplier.UpdatedAt = time.Now()
	if err := uc.repo.Update(supplier); err != nil {
		return nil, err
	}

	return toSupplierResponse(supplier), nil
}

func toSupplierResponse(s *entity.Supplier) *dto.SupplierResponse {
	if s == nil {
		return nil
	}
	return &dto.SupplierResponse{
		ID:              s.ID,
		CompanyID:       s.CompanyID,
		Name:            s.Name,
		NIT:             s.NIT,
		Email:           s.Email,
		Phone:           s.Phone,
		PaymentTermDays: s.PaymentTermDays,
		LeadTimeDays:    s.LeadTimeDays,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}
