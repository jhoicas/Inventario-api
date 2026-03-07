package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
)

// ── Fake DashboardUseCase (mock manual para tests) ──────────────────────────────

type fakeDashboardUseCase struct {
	getSummaryFunc func(ctx context.Context, companyID string) (*dto.DashboardSummaryDTO, error)
}

func (f *fakeDashboardUseCase) GetSummary(ctx context.Context, companyID string) (*dto.DashboardSummaryDTO, error) {
	if f.getSummaryFunc != nil {
		return f.getSummaryFunc(ctx, companyID)
	}
	return nil, errors.New("getSummary not configured")
}

// ── Helpers ────────────────────────────────────────────────────────────────────

const dashboardTestCompanyID = "company-dashboard-123"

// mockDashboardAuthMiddleware inyecta company_id en c.Locals para simular AuthMiddleware.
func mockDashboardAuthMiddleware(companyID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if companyID != "" {
			c.Locals(LocalCompanyID, companyID)
		}
		return c.Next()
	}
}

// validDashboardResponse genera un DashboardSummaryDTO de prueba coherente.
func validDashboardResponse() *dto.DashboardSummaryDTO {
	return &dto.DashboardSummaryDTO{
		TodaySales:    decimal.NewFromInt(150000),
		TodayMargin:   decimal.NewFromInt(45000),
		MonthlySales:  decimal.NewFromInt(2500000),
		MonthlyMargin: decimal.NewFromInt(750000),
		TopSKUs: []dto.TopSKUDTO{
			{
				ProductID:        "prod-001",
				SKU:              "SKU-001",
				ProductName:      "Producto A",
				QuantitySold:     decimal.NewFromInt(50),
				TotalRevenue:     decimal.NewFromInt(500000),
				MarginPercentage: decimal.NewFromFloat(30.5),
			},
		},
		DateLabel: "Febrero 2026",
	}
}

// ── Tests GetSummary ────────────────────────────────────────────────────────────
//
// Nota: El handler actual no recibe start_date/end_date por query params; las fechas
// se calculan en el servidor (hoy y mes en curso). Si en el futuro se añaden
// parámetros de período, se pueden agregar casos de prueba para su mapeo.

func TestDashboardHandler_GetSummary(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func() *fakeDashboardUseCase
		companyID      string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			mockSetup: func() *fakeDashboardUseCase {
				return &fakeDashboardUseCase{
					getSummaryFunc: func(_ context.Context, companyID string) (*dto.DashboardSummaryDTO, error) {
						assert.Equal(t, dashboardTestCompanyID, companyID)
						return validDashboardResponse(), nil
					},
				}
			},
			companyID:      dashboardTestCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.DashboardSummaryDTO
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.True(t, out.TodaySales.Equal(decimal.NewFromInt(150000)))
				assert.True(t, out.MonthlySales.Equal(decimal.NewFromInt(2500000)))
				assert.Equal(t, "Febrero 2026", out.DateLabel)
				require.Len(t, out.TopSKUs, 1)
				assert.Equal(t, "SKU-001", out.TopSKUs[0].SKU)
				assert.Equal(t, "Producto A", out.TopSKUs[0].ProductName)
			},
		},
		{
			name: "Unauthorized_NoCompanyID",
			mockSetup:      func() *fakeDashboardUseCase { return &fakeDashboardUseCase{} },
			companyID:      "",
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "UNAUTHORIZED", errResp.Code)
				assert.Contains(t, errResp.Message, "company_id")
			},
		},
		{
			name: "UseCase_InternalError",
			mockSetup: func() *fakeDashboardUseCase {
				return &fakeDashboardUseCase{
					getSummaryFunc: func(_ context.Context, _ string) (*dto.DashboardSummaryDTO, error) {
						return nil, errors.New("db connection failed")
					},
				}
			},
			companyID:      dashboardTestCompanyID,
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INTERNAL", errResp.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeUC := tt.mockSetup()
			handler := NewDashboardHandler(fakeUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			if tt.companyID != "" {
				app.Use(mockDashboardAuthMiddleware(tt.companyID))
			} else {
				app.Use(func(c *fiber.Ctx) error { return c.Next() })
			}
			app.Get("/dashboard/summary", handler.GetSummary)

			req := httptest.NewRequest(http.MethodGet, "/dashboard/summary", nil)

			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			if tt.validateBody != nil {
				tt.validateBody(t, resp)
			}
		})
	}
}
