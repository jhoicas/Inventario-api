package crm

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// AuditLogUseCase gestiona trazabilidad de cambios de entidades CRM/ERP.
type AuditLogUseCase struct {
	repo repository.AuditLogRepository
}

func NewAuditLogUseCase(repo repository.AuditLogRepository) *AuditLogUseCase {
	return &AuditLogUseCase{repo: repo}
}

func (uc *AuditLogUseCase) RegisterChange(ctx context.Context, companyID, userID, action, entityName, entityID string, before, after any) error {
	if uc == nil || uc.repo == nil {
		return nil
	}
	if strings.TrimSpace(companyID) == "" || strings.TrimSpace(action) == "" || strings.TrimSpace(entityName) == "" || strings.TrimSpace(entityID) == "" {
		return domain.ErrInvalidInput
	}
	payload := map[string]any{"before": before, "after": after}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	logEntry := &entity.AuditLog{
		ID:         uuid.New().String(),
		CompanyID:  companyID,
		UserID:     strings.TrimSpace(userID),
		Action:     strings.ToUpper(strings.TrimSpace(action)),
		EntityName: strings.ToUpper(strings.TrimSpace(entityName)),
		EntityID:   entityID,
		Changes:    raw,
		CreatedAt:  time.Now().UTC(),
	}
	return uc.repo.Create(ctx, logEntry)
}

func (uc *AuditLogUseCase) List(ctx context.Context, filters repository.AuditLogFilters) (*dto.AuditLogListResponse, error) {
	if uc == nil || uc.repo == nil || strings.TrimSpace(filters.CompanyID) == "" {
		return nil, domain.ErrInvalidInput
	}
	items, total, err := uc.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}
	metrics, err := uc.repo.CountByEntity(ctx, filters)
	if err != nil {
		return nil, err
	}
	respItems := make([]dto.AuditLogResponse, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, dto.AuditLogResponse{
			ID:         item.ID,
			CompanyID:  item.CompanyID,
			UserID:     item.UserID,
			Action:     item.Action,
			EntityName: item.EntityName,
			EntityID:   item.EntityID,
			Changes:    item.Changes,
			CreatedAt:  item.CreatedAt,
		})
	}
	respMetrics := make([]dto.AuditLogMetricByEntity, 0, len(metrics))
	for _, m := range metrics {
		respMetrics = append(respMetrics, dto.AuditLogMetricByEntity{EntityName: m.EntityName, Count: m.Count})
	}
	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}
	return &dto.AuditLogListResponse{
		Items:   respItems,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		Metrics: respMetrics,
	}, nil
}
