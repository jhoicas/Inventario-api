package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	appcrm "github.com/jhoicas/Inventario-api/internal/application/crm"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeNotificationLogRepo struct {
	items       []*entity.NotificationLog
	total       int64
	types       []string
	lastFilters repository.NotificationLogFilters
}

func (f *fakeNotificationLogRepo) Create(ctx context.Context, log *entity.NotificationLog) error {
	return nil
}

func (f *fakeNotificationLogRepo) List(ctx context.Context, filters repository.NotificationLogFilters) ([]*entity.NotificationLog, int64, error) {
	f.lastFilters = filters
	return f.items, f.total, nil
}

func (f *fakeNotificationLogRepo) ListTypes(ctx context.Context, companyID string, startDate, endDate *time.Time) ([]string, error) {
	return f.types, nil
}

func TestCRMHandler_ListNotifications_WithFiltersAndTypes(t *testing.T) {
	customerName := "Juan Perez"
	customerEmail := "juan@example.com"
	customerPhone := "+573001112233"
	repo := &fakeNotificationLogRepo{
		items: []*entity.NotificationLog{
			{
				ID:         "n1",
				CompanyID:  "comp-1",
				CustomerID: "cust-1",
				CustomerName: &customerName,
				CustomerEmail: &customerEmail,
				CustomerPhone: &customerPhone,
				Type:       "BIRTHDAY",
				Channel:    "EMAIL",
				Subject:    "Feliz cumpleaños",
				Body:       "<p>Hola</p>",
				SentAt:     time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC),
				Status:     "SENT",
			},
		},
		total: 1,
		types: []string{"campaign", "BIRTHDAY", "CAMPAIGN"},
	}

	uc := appcrm.NewNotificationLogUseCase(repo)
	h := &CRMHandler{NotificationUC: uc}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(LocalCompanyID, "comp-1")
		return c.Next()
	})
	app.Get("/api/crm/notifications", h.ListNotifications)

	req := httptest.NewRequest(http.MethodGet, "/api/crm/notifications?type=BIRTHDAY&start_date=2026-04-01T00:00:00Z&end_date=2026-04-30T23:59:59Z&limit=10&offset=5", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var out struct {
		Items  []map[string]any `json:"items"`
		Total  int64            `json:"total"`
		Limit  int              `json:"limit"`
		Offset int              `json:"offset"`
		Types  []string         `json:"types"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))

	assert.Len(t, out.Items, 1)
	assert.Equal(t, int64(1), out.Total)
	assert.Equal(t, 10, out.Limit)
	assert.Equal(t, 5, out.Offset)
	assert.Equal(t, []string{"BIRTHDAY", "CAMPAIGN"}, out.Types)
	assert.Equal(t, "Juan Perez", out.Items[0]["customer_name"])
	assert.Equal(t, "juan@example.com", out.Items[0]["customer_email"])
	assert.Equal(t, "+573001112233", out.Items[0]["customer_phone"])

	assert.Equal(t, "comp-1", repo.lastFilters.CompanyID)
	assert.Equal(t, "BIRTHDAY", repo.lastFilters.Type)
	require.NotNil(t, repo.lastFilters.StartDate)
	require.NotNil(t, repo.lastFilters.EndDate)
}

func TestCRMHandler_ListNotifications_NullCustomerEnrichment(t *testing.T) {
	repo := &fakeNotificationLogRepo{
		items: []*entity.NotificationLog{
			{
				ID:         "n-null",
				CompanyID:  "comp-1",
				CustomerID: "",
				Type:       "SYSTEM",
				Channel:    "EMAIL",
				Subject:    "System notice",
				Body:       "body",
				SentAt:     time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC),
				Status:     "SENT",
			},
		},
		total: 1,
		types: []string{"SYSTEM"},
	}
	uc := appcrm.NewNotificationLogUseCase(repo)
	h := &CRMHandler{NotificationUC: uc}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(LocalCompanyID, "comp-1")
		return c.Next()
	})
	app.Get("/api/crm/notifications", h.ListNotifications)

	req := httptest.NewRequest(http.MethodGet, "/api/crm/notifications", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var out struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	require.Len(t, out.Items, 1)
	assert.Nil(t, out.Items[0]["customer_name"])
	assert.Nil(t, out.Items[0]["customer_email"])
	assert.Nil(t, out.Items[0]["customer_phone"])
}

func TestCRMHandler_ListNotifications_DefaultPaging(t *testing.T) {
	repo := &fakeNotificationLogRepo{
		items: []*entity.NotificationLog{},
		total: 0,
		types: []string{},
	}
	uc := appcrm.NewNotificationLogUseCase(repo)
	h := &CRMHandler{NotificationUC: uc}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(LocalCompanyID, "comp-1")
		return c.Next()
	})
	app.Get("/api/crm/notifications", h.ListNotifications)

	req := httptest.NewRequest(http.MethodGet, "/api/crm/notifications", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var out struct {
		Total  int64    `json:"total"`
		Limit  int      `json:"limit"`
		Offset int      `json:"offset"`
		Types  []string `json:"types"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	assert.Equal(t, int64(0), out.Total)
	assert.Equal(t, 50, out.Limit)
	assert.Equal(t, 0, out.Offset)
	assert.Empty(t, out.Types)
}
