package crm

import (
	"context"
	"sort"
	"strings"
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
	types, err := uc.repo.ListTypes(ctx, companyID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	types = normalizeNotificationTypes(types)

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
		Types:  types,
	}, nil
}

func normalizeNotificationTypes(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		t := strings.ToUpper(strings.TrimSpace(raw))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
