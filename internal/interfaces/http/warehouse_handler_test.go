package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
)

// ── Fake WarehouseUseCase (mock manual para tests) ──────────────────────────────

type fakeWarehouseUseCase struct {
	createFunc func(companyID string, in dto.CreateWarehouseRequest) (*dto.WarehouseResponse, error)
	getByIDFunc func(id string) (*dto.WarehouseResponse, error)
	listFunc   func(companyID string, limit, offset int) (*dto.WarehouseListResponse, error)
}

func (f *fakeWarehouseUseCase) Create(companyID string, in dto.CreateWarehouseRequest) (*dto.WarehouseResponse, error) {
	if f.createFunc != nil {
		return f.createFunc(companyID, in)
	}
	return nil, errors.New("create not configured")
}

func (f *fakeWarehouseUseCase) GetByID(id string) (*dto.WarehouseResponse, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(id)
	}
	return nil, errors.New("getByID not configured")
}

func (f *fakeWarehouseUseCase) List(companyID string, limit, offset int) (*dto.WarehouseListResponse, error) {
	if f.listFunc != nil {
		return f.listFunc(companyID, limit, offset)
	}
	return nil, errors.New("list not configured")
}

// ── Helpers ────────────────────────────────────────────────────────────────────

const warehouseTestCompanyID = "company-warehouse-123"

// mockWarehouseAuthMiddleware inyecta company_id en c.Locals para simular AuthMiddleware.
func mockWarehouseAuthMiddleware(companyID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if companyID != "" {
			c.Locals(LocalCompanyID, companyID)
		}
		return c.Next()
	}
}

func validCreateWarehouseRequest() dto.CreateWarehouseRequest {
	return dto.CreateWarehouseRequest{
		Name:    "Bodega Central",
		Address: "Calle 123 #45-67",
	}
}

func validWarehouseResponse() *dto.WarehouseResponse {
	now := time.Now()
	return &dto.WarehouseResponse{
		ID:        "wh-001",
		CompanyID: warehouseTestCompanyID,
		Name:      "Bodega Central",
		Address:   "Calle 123 #45-67",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ── Tests Create ────────────────────────────────────────────────────────────────

func TestWarehouseHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func() *fakeWarehouseUseCase
		companyID      string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			body: validCreateWarehouseRequest(),
			mockSetup: func() *fakeWarehouseUseCase {
				return &fakeWarehouseUseCase{
					createFunc: func(_ string, _ dto.CreateWarehouseRequest) (*dto.WarehouseResponse, error) {
						return validWarehouseResponse(), nil
					},
				}
			},
			companyID:      warehouseTestCompanyID,
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.WarehouseResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "wh-001", out.ID)
				assert.Equal(t, "Bodega Central", out.Name)
				assert.Equal(t, "Calle 123 #45-67", out.Address)
			},
		},
		{
			name:           "Unauthorized_NoCompanyID",
			body:           validCreateWarehouseRequest(),
			mockSetup:      func() *fakeWarehouseUseCase { return &fakeWarehouseUseCase{} },
			companyID:      "",
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "UNAUTHORIZED", errResp.Code)
			},
		},
		{
			name:           "InvalidBody",
			body:           "not valid json",
			mockSetup:      func() *fakeWarehouseUseCase { return &fakeWarehouseUseCase{} },
			companyID:      warehouseTestCompanyID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INVALID_BODY", errResp.Code)
			},
		},
		{
			name: "Validation_NameRequired",
			body: dto.CreateWarehouseRequest{Name: "", Address: "Calle 1"},
			mockSetup:      func() *fakeWarehouseUseCase { return &fakeWarehouseUseCase{} },
			companyID:      warehouseTestCompanyID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "VALIDATION", errResp.Code)
				assert.Contains(t, errResp.Message, "name")
			},
		},
		{
			name: "UseCase_InternalError",
			body: validCreateWarehouseRequest(),
			mockSetup: func() *fakeWarehouseUseCase {
				return &fakeWarehouseUseCase{
					createFunc: func(_ string, _ dto.CreateWarehouseRequest) (*dto.WarehouseResponse, error) {
						return nil, errors.New("db error")
					},
				}
			},
			companyID:      warehouseTestCompanyID,
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
			handler := NewWarehouseHandler(fakeUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			if tt.companyID != "" {
				app.Use(mockWarehouseAuthMiddleware(tt.companyID))
			} else {
				app.Use(func(c *fiber.Ctx) error { return c.Next() })
			}
			app.Post("/warehouses", handler.Create)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/warehouses", bytes.NewReader(bodyBytes))
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

// ── Tests GetByID ───────────────────────────────────────────────────────────────

func TestWarehouseHandler_GetByID(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockSetup      func() *fakeWarehouseUseCase
		companyID      string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			id:   "wh-001",
			mockSetup: func() *fakeWarehouseUseCase {
				return &fakeWarehouseUseCase{
					getByIDFunc: func(id string) (*dto.WarehouseResponse, error) {
						return validWarehouseResponse(), nil
					},
				}
			},
			companyID:      warehouseTestCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.WarehouseResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "wh-001", out.ID)
				assert.Equal(t, "Bodega Central", out.Name)
			},
		},
		{
			name:           "BadRequest_MissingID",
			id:             "",
			mockSetup:      func() *fakeWarehouseUseCase { return &fakeWarehouseUseCase{} },
			companyID:      warehouseTestCompanyID,
			// Ruta sin :id no hace match en Fiber -> 404 plano.
			expectedStatus: http.StatusNotFound,
			validateBody:   nil,
		},
		{
			name: "NotFound",
			id:   "wh-999",
			mockSetup: func() *fakeWarehouseUseCase {
				return &fakeWarehouseUseCase{
					getByIDFunc: func(_ string) (*dto.WarehouseResponse, error) {
						return nil, nil
					},
				}
			},
			companyID:      warehouseTestCompanyID,
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_FOUND", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			id:   "wh-001",
			mockSetup: func() *fakeWarehouseUseCase {
				return &fakeWarehouseUseCase{
					getByIDFunc: func(_ string) (*dto.WarehouseResponse, error) {
						return nil, errors.New("db error")
					},
				}
			},
			companyID:      warehouseTestCompanyID,
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
			handler := NewWarehouseHandler(fakeUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockWarehouseAuthMiddleware(tt.companyID))
			app.Get("/warehouses/:id", handler.GetByID)

			path := "/warehouses/"
			if tt.id != "" {
				path += tt.id
			} else {
				path = "/warehouses//" // id vacío: Params("id") = ""
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

// ── Tests List ─────────────────────────────────────────────────────────────────

func TestWarehouseHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		mockSetup      func() *fakeWarehouseUseCase
		companyID      string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name:  "Success_DefaultPagination",
			query: "",
			mockSetup: func() *fakeWarehouseUseCase {
				return &fakeWarehouseUseCase{
					listFunc: func(companyID string, limit, offset int) (*dto.WarehouseListResponse, error) {
						assert.Equal(t, warehouseTestCompanyID, companyID)
						assert.Equal(t, 20, limit)
						assert.Equal(t, 0, offset)
						return &dto.WarehouseListResponse{
							Items: []dto.WarehouseResponse{*validWarehouseResponse()},
							Page:  dto.PageResponse{Limit: 20, Offset: 0},
						}, nil
					},
				}
			},
			companyID:      warehouseTestCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.WarehouseListResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Len(t, out.Items, 1)
				assert.Equal(t, "wh-001", out.Items[0].ID)
				assert.Equal(t, 20, out.Page.Limit)
				assert.Equal(t, 0, out.Page.Offset)
			},
		},
		{
			name:  "Success_WithPagination",
			query: "?limit=10&offset=5",
			mockSetup: func() *fakeWarehouseUseCase {
				return &fakeWarehouseUseCase{
					listFunc: func(companyID string, limit, offset int) (*dto.WarehouseListResponse, error) {
						assert.Equal(t, warehouseTestCompanyID, companyID)
						assert.Equal(t, 10, limit)
						assert.Equal(t, 5, offset)
						return &dto.WarehouseListResponse{
							Items: []dto.WarehouseResponse{},
							Page:  dto.PageResponse{Limit: 10, Offset: 5},
						}, nil
					},
				}
			},
			companyID:      warehouseTestCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.WarehouseListResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Len(t, out.Items, 0)
				assert.Equal(t, 10, out.Page.Limit)
				assert.Equal(t, 5, out.Page.Offset)
			},
		},
		{
			name:           "Unauthorized_NoCompanyID",
			query:          "",
			mockSetup:      func() *fakeWarehouseUseCase { return &fakeWarehouseUseCase{} },
			companyID:      "",
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "UNAUTHORIZED", errResp.Code)
			},
		},
		{
			name:  "UseCase_InternalError",
			query: "",
			mockSetup: func() *fakeWarehouseUseCase {
				return &fakeWarehouseUseCase{
					listFunc: func(_ string, _, _ int) (*dto.WarehouseListResponse, error) {
						return nil, errors.New("db error")
					},
				}
			},
			companyID:      warehouseTestCompanyID,
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
			handler := NewWarehouseHandler(fakeUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			if tt.companyID != "" {
				app.Use(mockWarehouseAuthMiddleware(tt.companyID))
			} else {
				app.Use(func(c *fiber.Ctx) error { return c.Next() })
			}
			app.Get("/warehouses", handler.List)

			req := httptest.NewRequest(http.MethodGet, "/warehouses"+tt.query, nil)

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
