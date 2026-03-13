package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// ── Fakes para los casos de uso de inventario ──────────────────────────────────

type fakeRegisterMovementUseCase struct {
	registerFunc func(ctx context.Context, companyID, userID string, in dto.RegisterMovementRequest) error
}

func (f *fakeRegisterMovementUseCase) RegisterMovementFromRequest(ctx context.Context, companyID, userID string, in dto.RegisterMovementRequest) error {
	if f.registerFunc != nil {
		return f.registerFunc(ctx, companyID, userID, in)
	}
	return errors.New("RegisterMovementFromRequest not configured")
}

type fakeReplenishmentUseCase struct {
	generateFunc func(ctx context.Context, companyID, warehouseID string) ([]dto.ReplenishmentSuggestionDTO, error)
}

func (f *fakeReplenishmentUseCase) GenerateReplenishmentList(ctx context.Context, companyID, warehouseID string) ([]dto.ReplenishmentSuggestionDTO, error) {
	if f.generateFunc != nil {
		return f.generateFunc(ctx, companyID, warehouseID)
	}
	return nil, errors.New("GenerateReplenishmentList not configured")
}

type fakeGetStockUseCase struct {
	executeFunc func(ctx context.Context, companyID, productID, warehouseID string) (*dto.StockSummaryDTO, error)
}

func (f *fakeGetStockUseCase) Execute(ctx context.Context, companyID, productID, warehouseID string) (*dto.StockSummaryDTO, error) {
	if f.executeFunc != nil {
		return f.executeFunc(ctx, companyID, productID, warehouseID)
	}
	return nil, errors.New("Execute not configured")
}

// ── Helpers ────────────────────────────────────────────────────────────────────

const inventoryTestCompanyID = "company-inv-123"
const inventoryTestUserID = "user-inv-456"

// mockInventoryAuthMiddleware simula AuthMiddleware inyectando company_id y user_id.
func mockInventoryAuthMiddleware(companyID, userID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if companyID != "" {
			c.Locals(LocalCompanyID, companyID)
		}
		if userID != "" {
			c.Locals(LocalUserID, userID)
		}
		return c.Next()
	}
}

// mockInventoryCompanyOnly inyecta solo company_id (para endpoints que no usan user_id).
func mockInventoryCompanyOnly(companyID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if companyID != "" {
			c.Locals(LocalCompanyID, companyID)
		}
		return c.Next()
	}
}

func validRegisterMovementRequest() dto.RegisterMovementRequest {
	qty := decimal.NewFromInt(10)
	unitCost := decimal.NewFromInt(5000)
	return dto.RegisterMovementRequest{
		ProductID:   "prod-001",
		WarehouseID: "wh-001",
		Type:        "IN",
		Quantity:    qty,
		UnitCost:    &unitCost,
	}
}

func validReplenishmentList() []dto.ReplenishmentSuggestionDTO {
	return []dto.ReplenishmentSuggestionDTO{
		{
			ProductID:         "prod-001",
			SKU:               "SKU-001",
			ProductName:       "Producto 1",
			CurrentStock:      decimal.NewFromInt(5),
			ReorderPoint:      decimal.NewFromInt(10),
			IdealStock:        decimal.NewFromInt(15),
			SuggestedOrderQty: decimal.NewFromInt(10),
		},
	}
}

// ── Tests RegisterMovement ──────────────────────────────────────────────────────

func TestInventoryHandler_RegisterMovement(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase)
		companyID      string
		userID         string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			body: validRegisterMovementRequest(),
			mockSetup: func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase) {
				reg := &fakeRegisterMovementUseCase{
					registerFunc: func(_ context.Context, companyID, userID string, in dto.RegisterMovementRequest) error {
						assert.Equal(t, inventoryTestCompanyID, companyID)
						assert.Equal(t, inventoryTestUserID, userID)
						assert.Equal(t, "prod-001", in.ProductID)
						return nil
					},
				}
				return reg, &fakeReplenishmentUseCase{}
			},
			companyID:      inventoryTestCompanyID,
			userID:         inventoryTestUserID,
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out map[string]string
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "movimiento registrado", out["message"])
			},
		},
		{
			name: "Unauthorized_NoCompanyOrUser",
			body: validRegisterMovementRequest(),
			mockSetup: func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase) {
				return &fakeRegisterMovementUseCase{}, &fakeReplenishmentUseCase{}
			},
			companyID:      "",
			userID:         "",
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "UNAUTHORIZED", errResp.Code)
			},
		},
		{
			name: "InvalidBody",
			body: "not valid json",
			mockSetup: func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase) {
				return &fakeRegisterMovementUseCase{}, &fakeReplenishmentUseCase{}
			},
			companyID:      inventoryTestCompanyID,
			userID:         inventoryTestUserID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INVALID_BODY", errResp.Code)
			},
		},
		{
			name: "Validation_ErrInvalidInput",
			body: validRegisterMovementRequest(),
			mockSetup: func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase) {
				reg := &fakeRegisterMovementUseCase{
					registerFunc: func(_ context.Context, _, _ string, _ dto.RegisterMovementRequest) error {
						return domain.ErrInvalidInput
					},
				}
				return reg, &fakeReplenishmentUseCase{}
			},
			companyID:      inventoryTestCompanyID,
			userID:         inventoryTestUserID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "VALIDATION", errResp.Code)
			},
		},
		{
			name: "NotFound_ProductOrWarehouse",
			body: validRegisterMovementRequest(),
			mockSetup: func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase) {
				reg := &fakeRegisterMovementUseCase{
					registerFunc: func(_ context.Context, _, _ string, _ dto.RegisterMovementRequest) error {
						return domain.ErrNotFound
					},
				}
				return reg, &fakeReplenishmentUseCase{}
			},
			companyID:      inventoryTestCompanyID,
			userID:         inventoryTestUserID,
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_FOUND", errResp.Code)
			},
		},
		{
			name: "Forbidden_AccessDenied",
			body: validRegisterMovementRequest(),
			mockSetup: func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase) {
				reg := &fakeRegisterMovementUseCase{
					registerFunc: func(_ context.Context, _, _ string, _ dto.RegisterMovementRequest) error {
						return domain.ErrForbidden
					},
				}
				return reg, &fakeReplenishmentUseCase{}
			},
			companyID:      inventoryTestCompanyID,
			userID:         inventoryTestUserID,
			expectedStatus: http.StatusForbidden,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "FORBIDDEN", errResp.Code)
			},
		},
		{
			name: "InsufficientStock_Conflict",
			body: validRegisterMovementRequest(),
			mockSetup: func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase) {
				reg := &fakeRegisterMovementUseCase{
					registerFunc: func(_ context.Context, _, _ string, _ dto.RegisterMovementRequest) error {
						return domain.ErrInsufficientStock
					},
				}
				return reg, &fakeReplenishmentUseCase{}
			},
			companyID:      inventoryTestCompanyID,
			userID:         inventoryTestUserID,
			expectedStatus: http.StatusConflict,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INSUFFICIENT_STOCK", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			body: validRegisterMovementRequest(),
			mockSetup: func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase) {
				reg := &fakeRegisterMovementUseCase{
					registerFunc: func(_ context.Context, _, _ string, _ dto.RegisterMovementRequest) error {
						return errors.New("db error")
					},
				}
				return reg, &fakeReplenishmentUseCase{}
			},
			companyID:      inventoryTestCompanyID,
			userID:         inventoryTestUserID,
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
			regUC, replUC := tt.mockSetup()
			handler := NewInventoryHandler(regUC, replUC, &fakeGetStockUseCase{})

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockInventoryAuthMiddleware(tt.companyID, tt.userID))
			app.Post("/inventory/movements", handler.RegisterMovement)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/inventory/movements", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

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

// ── Tests GetReplenishmentList ─────────────────────────────────────────────────

func TestInventoryHandler_GetReplenishmentList(t *testing.T) {
	tests := []struct {
		name           string
		warehouseID    string
		mockSetup      func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase)
		companyID      string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name:        "Success_GlobalStock",
			warehouseID: "",
			mockSetup: func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase) {
				repl := &fakeReplenishmentUseCase{
					generateFunc: func(_ context.Context, companyID, warehouseID string) ([]dto.ReplenishmentSuggestionDTO, error) {
						assert.Equal(t, inventoryTestCompanyID, companyID)
						assert.Equal(t, "", warehouseID)
						return validReplenishmentList(), nil
					},
				}
				return &fakeRegisterMovementUseCase{}, repl
			},
			companyID:      inventoryTestCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out struct {
					Total          int                              `json:"total"`
					Replenishments []dto.ReplenishmentSuggestionDTO `json:"replenishments"`
				}
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, 1, out.Total)
				require.Len(t, out.Replenishments, 1)
				assert.Equal(t, "prod-001", out.Replenishments[0].ProductID)
			},
		},
		{
			name:        "Success_FilterByWarehouse",
			warehouseID: "wh-001",
			mockSetup: func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase) {
				repl := &fakeReplenishmentUseCase{
					generateFunc: func(_ context.Context, companyID, warehouseID string) ([]dto.ReplenishmentSuggestionDTO, error) {
						assert.Equal(t, inventoryTestCompanyID, companyID)
						assert.Equal(t, "wh-001", warehouseID)
						return []dto.ReplenishmentSuggestionDTO{}, nil
					},
				}
				return &fakeRegisterMovementUseCase{}, repl
			},
			companyID:      inventoryTestCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out struct {
					Total          int                              `json:"total"`
					Replenishments []dto.ReplenishmentSuggestionDTO `json:"replenishments"`
				}
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, 0, out.Total)
				assert.Len(t, out.Replenishments, 0)
			},
		},
		{
			name:        "Unauthorized_NoCompanyID",
			warehouseID: "",
			mockSetup: func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase) {
				return &fakeRegisterMovementUseCase{}, &fakeReplenishmentUseCase{}
			},
			companyID:      "",
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "UNAUTHORIZED", errResp.Code)
			},
		},
		{
			name:        "UseCase_InternalError",
			warehouseID: "",
			mockSetup: func() (*fakeRegisterMovementUseCase, *fakeReplenishmentUseCase) {
				repl := &fakeReplenishmentUseCase{
					generateFunc: func(_ context.Context, _, _ string) ([]dto.ReplenishmentSuggestionDTO, error) {
						return nil, errors.New("db error")
					},
				}
				return &fakeRegisterMovementUseCase{}, repl
			},
			companyID:      inventoryTestCompanyID,
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
			regUC, replUC := tt.mockSetup()
			handler := NewInventoryHandler(regUC, replUC, &fakeGetStockUseCase{})

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockInventoryCompanyOnly(tt.companyID))
			app.Get("/inventory/replenishment-list", handler.GetReplenishmentList)

			path := "/inventory/replenishment-list"
			if tt.warehouseID != "" {
				path += "?warehouse_id=" + tt.warehouseID
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)

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

// ── Tests GetStock ───────────────────────────────────────────────────────────

func TestInventoryHandler_GetStock(t *testing.T) {
	lastUpdated := time.Date(2026, 3, 12, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name           string
		companyID      string
		productID      string
		warehouseID    string
		getStockUC     *fakeGetStockUseCase
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name:        "Success_WithWarehouse",
			companyID:   inventoryTestCompanyID,
			productID:   "prod-001",
			warehouseID: "wh-001",
			getStockUC: &fakeGetStockUseCase{
				executeFunc: func(_ context.Context, companyID, productID, warehouseID string) (*dto.StockSummaryDTO, error) {
					assert.Equal(t, inventoryTestCompanyID, companyID)
					assert.Equal(t, "prod-001", productID)
					assert.Equal(t, "wh-001", warehouseID)
					return &dto.StockSummaryDTO{
						ProductID:      productID,
						WarehouseID:    warehouseID,
						CurrentStock:   decimal.NewFromInt(25),
						ReservedStock:  decimal.NewFromInt(5),
						AvailableStock: decimal.NewFromInt(20),
						AvgCost:        decimal.NewFromInt(12500),
						LastUpdated:    lastUpdated,
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.StockSummaryDTO
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "prod-001", out.ProductID)
				assert.Equal(t, "wh-001", out.WarehouseID)
				assert.True(t, out.CurrentStock.Equal(decimal.NewFromInt(25)))
				assert.True(t, out.ReservedStock.Equal(decimal.NewFromInt(5)))
				assert.True(t, out.AvailableStock.Equal(decimal.NewFromInt(20)))
				assert.True(t, out.AvgCost.Equal(decimal.NewFromInt(12500)))
			},
		},
		{
			name:           "Unauthorized_NoCompanyID",
			companyID:      "",
			productID:      "prod-001",
			warehouseID:    "",
			getStockUC:     &fakeGetStockUseCase{},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "UNAUTHORIZED", errResp.Code)
			},
		},
		{
			name:           "Validation_MissingProductID",
			companyID:      inventoryTestCompanyID,
			productID:      "",
			warehouseID:    "",
			getStockUC:     &fakeGetStockUseCase{},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "VALIDATION", errResp.Code)
			},
		},
		{
			name:        "NotFound",
			companyID:   inventoryTestCompanyID,
			productID:   "prod-404",
			warehouseID: "wh-001",
			getStockUC: &fakeGetStockUseCase{
				executeFunc: func(_ context.Context, _, _, _ string) (*dto.StockSummaryDTO, error) {
					return nil, domain.ErrNotFound
				},
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_FOUND", errResp.Code)
			},
		},
		{
			name:        "InternalError",
			companyID:   inventoryTestCompanyID,
			productID:   "prod-001",
			warehouseID: "",
			getStockUC: &fakeGetStockUseCase{
				executeFunc: func(_ context.Context, _, _, _ string) (*dto.StockSummaryDTO, error) {
					return nil, errors.New("db error")
				},
			},
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
			handler := NewInventoryHandler(&fakeRegisterMovementUseCase{}, &fakeReplenishmentUseCase{}, tt.getStockUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockInventoryCompanyOnly(tt.companyID))
			app.Get("/inventory/stock", handler.GetStock)

			path := "/inventory/stock"
			params := make([]string, 0, 2)
			if tt.productID != "" {
				params = append(params, "product_id="+tt.productID)
			}
			if tt.warehouseID != "" {
				params = append(params, "warehouse_id="+tt.warehouseID)
			}
			if len(params) > 0 {
				path += "?" + strings.Join(params, "&")
			}

			req := httptest.NewRequest(http.MethodGet, path, nil)
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
