package crm

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
)

// OpportunityUseCase gestión de oportunidades CRM.
type OpportunityUseCase struct {
	oppRepo repository.CRMOpportunityRepository
}

// NewOpportunityUseCase construye el caso de uso.
func NewOpportunityUseCase(oppRepo repository.CRMOpportunityRepository) *OpportunityUseCase {
	return &OpportunityUseCase{oppRepo: oppRepo}
}

// Create crea una oportunidad en estado inicial (prospecto por defecto).
func (uc *OpportunityUseCase) Create(ctx context.Context, companyID, userID string, in dto.CreateOpportunityRequest) (*dto.OpportunityResponse, error) {
	if in.Title == "" {
		return nil, domain.ErrInvalidInput
	}

	stage := entity.OpportunityStage(in.Stage)
	if !isValidOpportunityStage(stage) {
		stage = entity.OpportunityStageProspecto
	}

	now := time.Now()
	opp := &entity.Opportunity{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		CustomerID:  in.CustomerID,
		Title:       in.Title,
		Amount:      in.Amount,
		Probability: in.Probability,
		Stage:       stage,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if in.ExpectedCloseDate != nil {
		opp.ExpectedCloseDate = *in.ExpectedCloseDate
	}

	if err := uc.oppRepo.Create(ctx, opp); err != nil {
		return nil, err
	}
	return toOpportunityResponse(opp), nil
}

// UpdateStage actualiza la etapa de una oportunidad.
func (uc *OpportunityUseCase) UpdateStage(ctx context.Context, companyID, id, stage string) error {
	if id == "" || stage == "" {
		return domain.ErrInvalidInput
	}

	s := entity.OpportunityStage(stage)
	if !isValidOpportunityStage(s) {
		return domain.ErrInvalidInput
	}

	opp, err := uc.oppRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if opp == nil {
		return domain.ErrNotFound
	}
	if opp.CompanyID != companyID {
		return domain.ErrForbidden
	}

	return uc.oppRepo.UpdateStage(ctx, id, s, time.Now())
}

// GetFunnel retorna el resumen por etapa del embudo de ventas de la empresa.
func (uc *OpportunityUseCase) GetFunnel(ctx context.Context, companyID string) ([]dto.FunnelStageDTO, error) {
	opps, err := uc.oppRepo.ListByCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}

	// Preservar orden del embudo.
	order := []entity.OpportunityStage{
		entity.OpportunityStageProspecto,
		entity.OpportunityStageCalificado,
		entity.OpportunityStagePropuesta,
		entity.OpportunityStageNegociacion,
		entity.OpportunityStageGanado,
		entity.OpportunityStagePerdido,
	}

	counts := make(map[entity.OpportunityStage]int, len(order))
	totals := make(map[entity.OpportunityStage]decimal.Decimal, len(order))
	for _, st := range order {
		counts[st] = 0
		totals[st] = decimal.Zero
	}

	for _, o := range opps {
		counts[o.Stage]++
		totals[o.Stage] = totals[o.Stage].Add(o.Amount)
	}

	result := make([]dto.FunnelStageDTO, 0, len(order))
	for _, st := range order {
		result = append(result, dto.FunnelStageDTO{
			Stage:       string(st),
			Count:       counts[st],
			TotalAmount: totals[st],
		})
	}
	return result, nil
}

// ListByCompany lista oportunidades de la empresa con paginación básica.
func (uc *OpportunityUseCase) ListByCompany(ctx context.Context, companyID string, limit, offset int) ([]dto.OpportunityResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	opps, err := uc.oppRepo.ListByCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if offset >= len(opps) {
		return []dto.OpportunityResponse{}, nil
	}
	end := offset + limit
	if end > len(opps) {
		end = len(opps)
	}
	out := make([]dto.OpportunityResponse, 0, end-offset)
	for _, o := range opps[offset:end] {
		resp := toOpportunityResponse(o)
		out = append(out, *resp)
	}
	return out, nil
}

func isValidOpportunityStage(s entity.OpportunityStage) bool {
	switch s {
	case entity.OpportunityStageProspecto,
		entity.OpportunityStageCalificado,
		entity.OpportunityStagePropuesta,
		entity.OpportunityStageNegociacion,
		entity.OpportunityStageGanado,
		entity.OpportunityStagePerdido:
		return true
	default:
		return false
	}
}

func toOpportunityResponse(o *entity.Opportunity) *dto.OpportunityResponse {
	resp := &dto.OpportunityResponse{
		ID:          o.ID,
		CompanyID:   o.CompanyID,
		CustomerID:  o.CustomerID,
		Title:       o.Title,
		Amount:      o.Amount,
		Probability: o.Probability,
		Stage:       string(o.Stage),
		CreatedBy:   o.CreatedBy,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	}
	if !o.ExpectedCloseDate.IsZero() {
		resp.ExpectedCloseDate = &o.ExpectedCloseDate
	}
	return resp
}
