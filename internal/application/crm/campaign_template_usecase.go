package crm

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// CampaignTemplateUseCase gestiona plantillas de campañas CRM.
type CampaignTemplateUseCase struct {
	repo repository.CRMCampaignTemplateRepository
}

func NewCampaignTemplateUseCase(repo repository.CRMCampaignTemplateRepository) *CampaignTemplateUseCase {
	return &CampaignTemplateUseCase{repo: repo}
}

func (uc *CampaignTemplateUseCase) CreateTemplate(ctx context.Context, companyID string, req dto.CreateCampaignTemplateRequest) (*dto.CampaignTemplateResponse, error) {
	if companyID == "" || req.Name == "" || req.Subject == "" || req.Body == "" {
		return nil, domain.ErrInvalidInput
	}
	now := time.Now()
	t := &entity.CampaignTemplate{
		ID:        uuid.New().String(),
		CompanyID: companyID,
		Name:      req.Name,
		Subject:   req.Subject,
		Body:      req.Body,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return &dto.CampaignTemplateResponse{
		ID:        t.ID,
		CompanyID: t.CompanyID,
		Name:      t.Name,
		Subject:   t.Subject,
		Body:      t.Body,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}, nil
}

func (uc *CampaignTemplateUseCase) GetTemplates(ctx context.Context, companyID string) ([]dto.CampaignTemplateResponse, error) {
	if companyID == "" {
		return nil, domain.ErrInvalidInput
	}
	list, err := uc.repo.FindAllByCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.CampaignTemplateResponse, 0, len(list))
	for _, t := range list {
		out = append(out, dto.CampaignTemplateResponse{
			ID:        t.ID,
			CompanyID: t.CompanyID,
			Name:      t.Name,
			Subject:   t.Subject,
			Body:      t.Body,
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		})
	}
	return out, nil
}

func (uc *CampaignTemplateUseCase) DeleteTemplate(ctx context.Context, companyID, id string) error {
	if companyID == "" || id == "" {
		return domain.ErrInvalidInput
	}
	return uc.repo.Delete(ctx, id, companyID)
}

