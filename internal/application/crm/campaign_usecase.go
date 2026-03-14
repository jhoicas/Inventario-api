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

// CampaignUseCase gestión de campañas de marketing CRM.
type CampaignUseCase struct {
	repo repository.CRMCampaignRepository
}

// NewCampaignUseCase construye el caso de uso.
func NewCampaignUseCase(repo repository.CRMCampaignRepository) *CampaignUseCase {
	return &CampaignUseCase{repo: repo}
}

// Create crea una campaña en estado BORRADOR.
func (uc *CampaignUseCase) Create(ctx context.Context, companyID, userID string, req dto.CreateCampaignRequest) (*dto.CampaignResponse, error) {
	if req.Name == "" {
		return nil, domain.ErrInvalidInput
	}

	now := time.Now()
	c := &entity.Campaign{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		Name:        req.Name,
		Description: req.Description,
		Status:      entity.CampaignStatusBorrador,
		ScheduledAt: req.ScheduledAt,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return toCampaignResponse(c), nil
}

// GetMetrics devuelve las métricas de envío y conversión de una campaña.
func (uc *CampaignUseCase) GetMetrics(ctx context.Context, campaignID string) (*dto.CampaignMetricsResponse, error) {
	if campaignID == "" {
		return nil, domain.ErrInvalidInput
	}

	m, err := uc.repo.GetMetrics(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, domain.ErrNotFound
	}

	return &dto.CampaignMetricsResponse{
		CampaignID: m.CampaignID,
		Sent:       m.Sent,
		Opened:     m.Opened,
		Clicked:    m.Clicked,
		Converted:  m.Converted,
		Revenue:    m.Revenue,
	}, nil
}

func toCampaignResponse(c *entity.Campaign) *dto.CampaignResponse {
	resp := &dto.CampaignResponse{
		ID:          c.ID,
		CompanyID:   c.CompanyID,
		Name:        c.Name,
		Description: c.Description,
		Status:      string(c.Status),
		CreatedBy:   c.CreatedBy,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
	if c.ScheduledAt != nil {
		resp.ScheduledAt = c.ScheduledAt
	}
	return resp
}
