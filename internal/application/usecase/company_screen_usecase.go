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

// CompanyScreenUseCase gestiona pantallas habilitadas por empresa.
type CompanyScreenUseCase struct {
	companyRepo repository.CompanyRepository
	rbacRepo    repository.RBACRepository
}

// NewCompanyScreenUseCase construye el caso de uso de company_screens.
func NewCompanyScreenUseCase(companyRepo repository.CompanyRepository, rbacRepo repository.RBACRepository) *CompanyScreenUseCase {
	return &CompanyScreenUseCase{companyRepo: companyRepo, rbacRepo: rbacRepo}
}

// ListScreens devuelve las pantallas configuradas para una empresa.
func (uc *CompanyScreenUseCase) ListScreens(ctx context.Context, companyID string) (*dto.CompanyScreensResponse, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}
	company, err := uc.companyRepo.GetByID(companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.ErrNotFound
	}
	list, err := uc.companyRepo.ListScreens(ctx, companyID)
	if err != nil {
		return nil, err
	}
	out := &dto.CompanyScreensResponse{
		CompanyID: companyID,
		Screens:   make([]dto.CompanyScreenResponse, 0, len(list)),
	}
	for _, item := range list {
		out.Screens = append(out.Screens, companyScreenToDTO(item))
	}
	return out, nil
}

// UpsertScreen crea o actualiza la configuración de una pantalla para la empresa.
func (uc *CompanyScreenUseCase) UpsertScreen(ctx context.Context, companyID string, in dto.CreateCompanyScreenRequest) (*dto.CompanyScreenResponse, error) {
	if companyID == "" || in.ScreenID == "" {
		return nil, domain.ErrInvalidInput
	}
	company, err := uc.companyRepo.GetByID(companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.ErrNotFound
	}
	screen, err := uc.rbacRepo.GetScreenByID(ctx, in.ScreenID)
	if err != nil {
		return nil, err
	}
	if screen == nil {
		return nil, domain.ErrNotFound
	}
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	now := time.Now()
	current, err := uc.companyRepo.GetScreen(ctx, companyID, in.ScreenID)
	if err != nil {
		return nil, err
	}
	model := &entity.CompanyScreen{
		ID:            uuid.New().String(),
		CompanyID:     companyID,
		ScreenID:      in.ScreenID,
		ScreenKey:     screen.Key,
		ScreenName:    screen.Name,
		ModuleKey:     screen.ModuleKey,
		ModuleName:    screen.ModuleName,
		FrontendRoute: screen.FrontendRoute,
		ApiEndpoint:   screen.ApiEndpoint,
		IsActive:      active,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if current != nil {
		model.ID = current.ID
		model.CreatedAt = current.CreatedAt
	}
	if err := uc.companyRepo.UpsertScreen(ctx, model); err != nil {
		return nil, err
	}
	updated, err := uc.companyRepo.GetScreen(ctx, companyID, in.ScreenID)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, domain.ErrNotFound
	}
	out := companyScreenToDTO(updated)
	return &out, nil
}

// UpdateScreen actualiza el estado de una pantalla existente para la empresa.
func (uc *CompanyScreenUseCase) UpdateScreen(ctx context.Context, companyID, screenID string, in dto.UpdateCompanyScreenRequest) (*dto.CompanyScreenResponse, error) {
	if companyID == "" || screenID == "" {
		return nil, domain.ErrInvalidInput
	}
	company, err := uc.companyRepo.GetByID(companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.ErrNotFound
	}
	screen, err := uc.rbacRepo.GetScreenByID(ctx, screenID)
	if err != nil {
		return nil, err
	}
	if screen == nil {
		return nil, domain.ErrNotFound
	}
	current, err := uc.companyRepo.GetScreen(ctx, companyID, screenID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, domain.ErrNotFound
	}
	if in.IsActive != nil {
		current.IsActive = *in.IsActive
	}
	current.ScreenKey = screen.Key
	current.ScreenName = screen.Name
	current.ModuleKey = screen.ModuleKey
	current.ModuleName = screen.ModuleName
	current.FrontendRoute = screen.FrontendRoute
	current.ApiEndpoint = screen.ApiEndpoint
	current.UpdatedAt = time.Now()
	if err := uc.companyRepo.UpsertScreen(ctx, current); err != nil {
		return nil, err
	}
	updated, err := uc.companyRepo.GetScreen(ctx, companyID, screenID)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, domain.ErrNotFound
	}
	out := companyScreenToDTO(updated)
	return &out, nil
}

// DeleteScreen desactiva la pantalla para la empresa.
func (uc *CompanyScreenUseCase) DeleteScreen(ctx context.Context, companyID, screenID string) error {
	if companyID == "" || screenID == "" {
		return domain.ErrInvalidInput
	}
	return uc.companyRepo.DeleteScreen(ctx, companyID, screenID)
}

// ReplaceScreens reemplaza en bloque las pantallas de una empresa.
func (uc *CompanyScreenUseCase) ReplaceScreens(ctx context.Context, companyID string, in dto.ReplaceCompanyScreensRequest) (*dto.CompanyScreensResponse, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}
	company, err := uc.companyRepo.GetByID(companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.ErrNotFound
	}

	for _, screenID := range in.ScreenIDs {
		if screenID == "" {
			return nil, domain.ErrInvalidInput
		}
		screen, err := uc.rbacRepo.GetScreenByID(ctx, screenID)
		if err != nil {
			return nil, err
		}
		if screen == nil {
			return nil, domain.ErrNotFound
		}
	}

	if err := uc.companyRepo.ReplaceScreens(ctx, companyID, in.ScreenIDs); err != nil {
		return nil, err
	}

	list, err := uc.companyRepo.ListScreens(ctx, companyID)
	if err != nil {
		return nil, err
	}
	out := &dto.CompanyScreensResponse{
		CompanyID: companyID,
		Screens:   make([]dto.CompanyScreenResponse, 0, len(list)),
	}
	for _, item := range list {
		out.Screens = append(out.Screens, companyScreenToDTO(item))
	}
	return out, nil
}

func companyScreenToDTO(in *entity.CompanyScreen) dto.CompanyScreenResponse {
	if in == nil {
		return dto.CompanyScreenResponse{}
	}
	createdAt := in.CreatedAt
	updatedAt := in.UpdatedAt
	return dto.CompanyScreenResponse{
		ID:            in.ID,
		CompanyID:     in.CompanyID,
		ScreenID:      in.ScreenID,
		ScreenKey:     in.ScreenKey,
		ScreenName:    in.ScreenName,
		ModuleKey:     in.ModuleKey,
		ModuleName:    in.ModuleName,
		FrontendRoute: in.FrontendRoute,
		ApiEndpoint:   in.ApiEndpoint,
		IsActive:      in.IsActive,
		CreatedAt:     &createdAt,
		UpdatedAt:     &updatedAt,
	}
}
