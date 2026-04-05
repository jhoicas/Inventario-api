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

// Update actualiza una empresa existente.
func (uc *CompanyUseCase) Update(id string, in dto.UpdateCompanyRequest) (*dto.CompanyResponse, error) {
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	company, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.ErrNotFound
	}

	if in.Name != nil {
		company.Name = *in.Name
	}
	if in.NIT != nil {
		company.NIT = *in.NIT
	}
	if in.Address != nil {
		company.Address = *in.Address
	}
	if in.Phone != nil {
		company.Phone = *in.Phone
	}
	if in.Email != nil {
		company.Email = *in.Email
	}
	if in.Status != nil {
		company.Status = *in.Status
	}

	company.UpdatedAt = time.Now()
	if err := uc.repo.Update(company); err != nil {
		return nil, err
	}
	return entityToCompanyResponse(company), nil
}

// Delete elimina una empresa.
func (uc *CompanyUseCase) Delete(id string) error {
	if id == "" {
		return domain.ErrInvalidInput
	}
	return uc.repo.Delete(id)
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

// ListForAdmin lista empresas para superadmin excluyendo la cuenta técnica it@ludoia.com.
func (uc *CompanyUseCase) ListForAdmin(limit, offset int) (*dto.CompanyListResponse, error) {
	list, err := uc.repo.ListForAdmin(limit, offset)
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
	// environment opcional: si no viene, usar "test" por defecto para compatibilidad.
	env := in.Environment
	if env == "" {
		env = "test"
	}
	if env != "test" && env != "prod" {
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
		Environment:      env,
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

// ListModules devuelve los módulos SaaS contratados por la empresa.
func (uc *CompanyUseCase) ListModules(ctx context.Context, companyID string) (*dto.CompanyModulesResponse, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}
	list, err := uc.repo.ListModules(ctx, companyID)
	if err != nil {
		return nil, err
	}
	out := &dto.CompanyModulesResponse{
		CompanyID: companyID,
		Modules:   make([]dto.CompanyModuleResponse, 0, len(list)),
	}
	for _, m := range list {
		activatedAt := m.ActivatedAt
		createdAt := m.CreatedAt
		updatedAt := m.UpdatedAt
		out.Modules = append(out.Modules, dto.CompanyModuleResponse{
			ID:          m.ID,
			ModuleName:  m.ModuleName,
			IsActive:    m.IsActive,
			ActivatedAt: &activatedAt,
			ExpiresAt:   m.ExpiresAt,
			CreatedAt:   &createdAt,
			UpdatedAt:   &updatedAt,
		})
	}
	return out, nil
}

// UpsertModule crea/actualiza un módulo para una empresa.
func (uc *CompanyUseCase) UpsertModule(ctx context.Context, companyID string, in dto.CreateCompanyModuleRequest) (*dto.CompanyModuleResponse, error) {
	if companyID == "" || in.ModuleName == "" {
		return nil, domain.ErrInvalidInput
	}
	company, err := uc.repo.GetByID(companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.ErrNotFound
	}

	now := time.Now()
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}

	var expiresAt *time.Time
	if in.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, in.ExpiresAt)
		if err != nil {
			return nil, domain.ErrInvalidInput
		}
		expiresAt = &parsed
	}

	current, err := uc.repo.GetModule(ctx, companyID, in.ModuleName)
	if err != nil {
		return nil, err
	}

	module := &entity.CompanyModule{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		ModuleName:  in.ModuleName,
		IsActive:    active,
		ActivatedAt: now,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if current != nil {
		module.ID = current.ID
		module.CreatedAt = current.CreatedAt
		module.ActivatedAt = current.ActivatedAt
		if active && !current.IsActive {
			module.ActivatedAt = now
		}
	}

	if err := uc.repo.UpsertModule(ctx, module); err != nil {
		return nil, err
	}

	createdAt := module.CreatedAt
	updatedAt := module.UpdatedAt
	activatedAt := module.ActivatedAt
	return &dto.CompanyModuleResponse{
		ID:          module.ID,
		ModuleName:  module.ModuleName,
		IsActive:    module.IsActive,
		ActivatedAt: &activatedAt,
		ExpiresAt:   module.ExpiresAt,
		CreatedAt:   &createdAt,
		UpdatedAt:   &updatedAt,
	}, nil
}

// UpdateModule actualiza datos de un módulo ya existente.
func (uc *CompanyUseCase) UpdateModule(ctx context.Context, companyID, moduleName string, in dto.UpdateCompanyModuleRequest) (*dto.CompanyModuleResponse, error) {
	if companyID == "" || moduleName == "" {
		return nil, domain.ErrInvalidInput
	}
	module, err := uc.repo.GetModule(ctx, companyID, moduleName)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, domain.ErrNotFound
	}

	if in.IsActive != nil {
		module.IsActive = *in.IsActive
		if module.IsActive {
			module.ActivatedAt = time.Now()
		}
	}
	if in.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, in.ExpiresAt)
		if err != nil {
			return nil, domain.ErrInvalidInput
		}
		module.ExpiresAt = &parsed
	}
	module.UpdatedAt = time.Now()

	if err := uc.repo.UpsertModule(ctx, module); err != nil {
		return nil, err
	}

	createdAt := module.CreatedAt
	updatedAt := module.UpdatedAt
	activatedAt := module.ActivatedAt
	return &dto.CompanyModuleResponse{
		ID:          module.ID,
		ModuleName:  module.ModuleName,
		IsActive:    module.IsActive,
		ActivatedAt: &activatedAt,
		ExpiresAt:   module.ExpiresAt,
		CreatedAt:   &createdAt,
		UpdatedAt:   &updatedAt,
	}, nil
}

// DeleteModule elimina un módulo de la empresa.
func (uc *CompanyUseCase) DeleteModule(ctx context.Context, companyID, moduleName string) error {
	if companyID == "" || moduleName == "" {
		return domain.ErrInvalidInput
	}
	return uc.repo.DeleteModule(ctx, companyID, moduleName)
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
