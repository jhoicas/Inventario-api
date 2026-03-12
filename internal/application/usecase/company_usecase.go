package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// CompanyUseCase aplica reglas de negocio para empresas (casos de uso).
type CompanyUseCase struct {
	repo           repository.CompanyRepository
	resolutionRepo repository.BillingResolutionRepository
}

// NewCompanyUseCase construye el caso de uso con el puerto de persistencia.
func NewCompanyUseCase(repo repository.CompanyRepository, resolutionRepo repository.BillingResolutionRepository) *CompanyUseCase {
	return &CompanyUseCase{repo: repo, resolutionRepo: resolutionRepo}
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

// CreateResolution crea una resolución DIAN para la empresa.
func (uc *CompanyUseCase) CreateResolution(companyID string, in dto.CreateResolutionRequest) (*dto.ResolutionResponse, error) {
	if companyID == "" || in.Prefix == "" || in.ResolutionNumber == "" || in.FromNumber <= 0 || in.ToNumber <= 0 || in.ToNumber < in.FromNumber {
		return nil, domain.ErrInvalidInput
	}
	if in.Environment != "test" && in.Environment != "prod" {
		return nil, domain.ErrInvalidInput
	}
	company, err := uc.repo.GetByID(companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.ErrNotFound
	}

	validFrom, err := time.Parse("2006-01-02", in.ValidFrom)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	validUntil, err := time.Parse("2006-01-02", in.ValidUntil)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	if validUntil.Before(validFrom) {
		return nil, domain.ErrInvalidInput
	}

	now := time.Now()
	res := &entity.BillingResolution{
		ID:               uuid.New().String(),
		CompanyID:        companyID,
		ResolutionNumber: in.ResolutionNumber,
		Prefix:           in.Prefix,
		RangeFrom:        in.FromNumber,
		RangeTo:          in.ToNumber,
		DateFrom:         validFrom,
		DateTo:           validUntil,
		Environment:      in.Environment,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := uc.resolutionRepo.Create(context.Background(), res); err != nil {
		return nil, err
	}

	return resolutionToDTO(res), nil
}

// ListResolutions lista resoluciones de una empresa con bandera de alert threshold.
func (uc *CompanyUseCase) ListResolutions(companyID string) ([]dto.ResolutionResponse, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}
	company, err := uc.repo.GetByID(companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.ErrNotFound
	}
	list, err := uc.resolutionRepo.ListByCompany(context.Background(), companyID)
	if err != nil {
		return nil, err
	}

	out := make([]dto.ResolutionResponse, 0, len(list))
	for _, res := range list {
		out = append(out, *resolutionToDTO(res))
	}
	return out, nil
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

func resolutionToDTO(res *entity.BillingResolution) *dto.ResolutionResponse {
	if res == nil {
		return nil
	}
	total := res.RangeTo - res.RangeFrom + 1
	if total < 0 {
		total = 0
	}
	used := res.UsedNumbers
	if used < 0 {
		used = 0
	}
	if used > total {
		used = total
	}
	available := total - used
	alert := total > 0 && float64(available) < float64(total)*0.10

	return &dto.ResolutionResponse{
		ID:               res.ID,
		CompanyID:        res.CompanyID,
		Prefix:           res.Prefix,
		ResolutionNumber: res.ResolutionNumber,
		FromNumber:       res.RangeFrom,
		ToNumber:         res.RangeTo,
		ValidFrom:        res.DateFrom,
		ValidUntil:       res.DateTo,
		Environment:      res.Environment,
		AlertThreshold:   alert,
		CreatedAt:        res.CreatedAt,
		UpdatedAt:        res.UpdatedAt,
	}
}
