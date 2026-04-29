package crm

import (
	"context"
	"time"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

type NotificationLogUseCase struct {
	repo repository.NotificationLogRepository
}

func NewNotificationLogUseCase(repo repository.NotificationLogRepository) *NotificationLogUseCase {
	return &NotificationLogUseCase{repo: repo}
}

func (uc *NotificationLogUseCase) List(
	ctx context.Context,
	companyID, typ string,
	startDate, endDate *time.Time,
	limit, offset int,
) (*dto.NotificationLogListResponse, error) {
	items, total, err := uc.repo.List(ctx, repository.NotificationLogFilters{
		CompanyID: companyID,
		Type:      typ,
		StartDate: startDate,
		EndDate:   endDate,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}

	out := make([]dto.NotificationLogResponse, 0, len(items))
	for _, item := range items {
		out = append(out, dto.NotificationLogResponse{
			ID:           item.ID,
			CompanyID:    item.CompanyID,
			CustomerID:   item.CustomerID,
			Type:         item.Type,
			Channel:      item.Channel,
			Subject:      item.Subject,
			Body:         item.Body,
			SentAt:       item.SentAt,
			Status:       item.Status,
			ErrorMessage: item.ErrorMessage,
		})
	}

	return &dto.NotificationLogListResponse{
		Items:  out,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}
