package crm

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// CampaignTemplateUseCase gestiona plantillas de campañas CRM.
type CampaignTemplateUseCase struct {
	repo    repository.CRMCampaignTemplateRepository
	auditUC *AuditLogUseCase
}

func NewCampaignTemplateUseCase(repo repository.CRMCampaignTemplateRepository, auditUC *AuditLogUseCase) *CampaignTemplateUseCase {
	return &CampaignTemplateUseCase{repo: repo, auditUC: auditUC}
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
	return uc.DeleteTemplateByUser(ctx, companyID, "", id)
}

func (uc *CampaignTemplateUseCase) DeleteTemplateByUser(ctx context.Context, companyID, userID, id string) error {
	if companyID == "" || id == "" {
		return domain.ErrInvalidInput
	}
	current, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if current == nil || current.CompanyID != companyID {
		return domain.ErrNotFound
	}
	if err := uc.repo.Delete(ctx, id, companyID); err != nil {
		return err
	}
	_ = uc.auditUC.RegisterChange(ctx, companyID, userID, "DELETE", "TEMPLATE", id, current, nil)
	return nil
}

func (uc *CampaignTemplateUseCase) UpdateTemplate(ctx context.Context, companyID, userID, id string, req dto.UpdateCampaignTemplateRequest) (*dto.CampaignTemplateResponse, error) {
	if strings.TrimSpace(companyID) == "" || strings.TrimSpace(id) == "" {
		return nil, domain.ErrInvalidInput
	}
	current, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil || current.CompanyID != companyID {
		return nil, domain.ErrNotFound
	}
	before := *current
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, domain.ErrInvalidInput
		}
		current.Name = name
	}
	if req.Subject != nil {
		subject := strings.TrimSpace(*req.Subject)
		if subject == "" {
			return nil, domain.ErrInvalidInput
		}
		current.Subject = subject
	}
	if req.Body != nil {
		body := strings.TrimSpace(*req.Body)
		if body == "" {
			return nil, domain.ErrInvalidInput
		}
		current.Body = body
	}
	current.UpdatedAt = time.Now()
	if err := uc.repo.Update(ctx, current); err != nil {
		return nil, err
	}
	_ = uc.auditUC.RegisterChange(ctx, companyID, userID, "UPDATE", "TEMPLATE", current.ID, before, current)
	return &dto.CampaignTemplateResponse{
		ID:        current.ID,
		CompanyID: current.CompanyID,
		Name:      current.Name,
		Subject:   current.Subject,
		Body:      current.Body,
		CreatedAt: current.CreatedAt,
		UpdatedAt: current.UpdatedAt,
	}, nil
}
