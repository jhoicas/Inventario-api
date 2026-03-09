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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// ── Fake ProductUseCase (mock manual para tests) ────────────────────────────────

type fakeProductUseCase struct {
	createFunc func(companyID string, in dto.CreateProductRequest) (*dto.ProductResponse, error)
	getByIDFunc func(id string) (*dto.ProductResponse, error)
	listFunc   func(companyID string, limit, offset int) (*dto.ProductListResponse, error)
	updateFunc func(id string, in dto.UpdateProductRequest) (*dto.ProductResponse, error)
}

func (f *fakeProductUseCase) Create(companyID string, in dto.CreateProductRequest) (*dto.ProductResponse, error) {
	if f.createFunc != nil {
		return f.createFunc(companyID, in)
	}
	return nil, errors.New("create not configured")
}

func (f *fakeProductUseCase) GetByID(id string) (*dto.ProductResponse, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(id)
	}
	return nil, errors.New("getByID not configured")
}

func (f *fakeProductUseCase) List(companyID string, limit, offset int) (*dto.ProductListResponse, error) {
	if f.listFunc != nil {
		return f.listFunc(companyID, limit, offset)
	}
	return nil, errors.New("list not configured")
}

func (f *fakeProductUseCase) Update(id string, in dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	if f.updateFunc != nil {
		return f.updateFunc(id, in)
	}
	return nil, errors.New("update not configured")
}

// ── Helpers ────────────────────────────────────────────────────────────────────

const testCompanyID = "company-123"

func mockCompanyMiddleware(companyID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(LocalCompanyID, companyID)
		return c.Next()
	}
}

func validCreateProduct() dto.CreateProductRequest {
	return dto.CreateProductRequest{
		SKU:         "SKU-001",
		Name:        "Producto de prueba",
		Description: "Descripción",
		Price:       decimal.NewFromInt(10000),
		TaxRate:     decimal.NewFromInt(19),
		UnitMeasure: "94",
	}
}

func validProductResponse() *dto.ProductResponse {
	return &dto.ProductResponse{
		ID:          "prod-123",
		CompanyID:   testCompanyID,
		SKU:         "SKU-001",
		Name:        "Producto de prueba",
		Description: "Descripción",
		Price:       decimal.NewFromInt(10000),
		Cost:        decimal.Zero,
		TaxRate:     decimal.NewFromInt(19),
		UNSPSC_Code: "",
		UnitMeasure: "94",
		Attributes:  nil,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ── Tests Create ────────────────────────────────────────────────────────────────

func TestProductHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func() *fakeProductUseCase
		companyID      string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			body: validCreateProduct(),
			mockSetup: func() *fakeProductUseCase {
				return &fakeProductUseCase{
					createFunc: func(_ string, _ dto.CreateProductRequest) (*dto.ProductResponse, error) {
						return validProductResponse(), nil
					},
				}
			},
			companyID:      testCompanyID,
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.ProductResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "prod-123", out.ID)
				assert.Equal(t, "SKU-001", out.SKU)
				assert.Equal(t, "Producto de prueba", out.Name)
			},
		},
		{
			name:           "Unauthorized_NoCompanyID",
			body:           validCreateProduct(),
			mockSetup:      func() *fakeProductUseCase { return &fakeProductUseCase{} },
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
			mockSetup:      func() *fakeProductUseCase { return &fakeProductUseCase{} },
			companyID:      testCompanyID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INVALID_BODY", errResp.Code)
			},
		},
		{
			name: "Validation_SKUAndNameRequired",
			body: dto.CreateProductRequest{
				SKU:         "",
				Name:        "",
				Price:       decimal.Zero,
				TaxRate:     decimal.Zero,
				UnitMeasure: "94",
			},
			mockSetup:      func() *fakeProductUseCase { return &fakeProductUseCase{} },
			companyID:      testCompanyID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "VALIDATION", errResp.Code)
				assert.Contains(t, errResp.Message, "sku")
			},
		},
		{
			name: "Duplicate_SKUExists",
			body: validCreateProduct(),
			mockSetup: func() *fakeProductUseCase {
				return &fakeProductUseCase{
					createFunc: func(_ string, _ dto.CreateProductRequest) (*dto.ProductResponse, error) {
						return nil, domain.ErrDuplicate
					},
				}
			},
			companyID:      testCompanyID,
			expectedStatus: http.StatusConflict,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "DUPLICATE", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			body: validCreateProduct(),
			mockSetup: func() *fakeProductUseCase {
				return &fakeProductUseCase{
					createFunc: func(_ string, _ dto.CreateProductRequest) (*dto.ProductResponse, error) {
						return nil, errors.New("db connection failed")
					},
				}
			},
			companyID:      testCompanyID,
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

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			if tt.companyID != "" {
				app.Use(mockCompanyMiddleware(tt.companyID))
			} else {
				app.Use(func(c *fiber.Ctx) error { return c.Next() })
			}
			app.Post("/products", NewProductHandler(fakeUC).Create)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(bodyBytes))
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

func TestProductHandler_GetByID(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockSetup      func() *fakeProductUseCase
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			id:   "prod-123",
			mockSetup: func() *fakeProductUseCase {
				return &fakeProductUseCase{
					getByIDFunc: func(id string) (*dto.ProductResponse, error) {
						return validProductResponse(), nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.ProductResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "prod-123", out.ID)
			},
		},
		{
			name:           "BadRequest_MissingID",
			id:             "",
			mockSetup:      func() *fakeProductUseCase { return &fakeProductUseCase{} },
			// Ruta sin :id no hace match, Fiber responde 404.
			expectedStatus: http.StatusNotFound,
			validateBody:   nil,
		},
		{
			name: "NotFound",
			id:   "prod-999",
			mockSetup: func() *fakeProductUseCase {
				return &fakeProductUseCase{
					getByIDFunc: func(_ string) (*dto.ProductResponse, error) {
						return nil, nil
					},
				}
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_FOUND", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			id:   "prod-123",
			mockSetup: func() *fakeProductUseCase {
				return &fakeProductUseCase{
					getByIDFunc: func(_ string) (*dto.ProductResponse, error) {
						return nil, errors.New("db error")
					},
				}
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
			fakeUC := tt.mockSetup()

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockCompanyMiddleware(testCompanyID))
			app.Get("/products/:id", NewProductHandler(fakeUC).GetByID)

			path := "/products/"
			if tt.id != "" {
				path += tt.id
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

// ── Tests List ──────────────────────────────────────────────────────────────────

func TestProductHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		mockSetup      func() *fakeProductUseCase
		companyID      string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name:  "Success",
			query: "",
			mockSetup: func() *fakeProductUseCase {
				return &fakeProductUseCase{
					listFunc: func(_ string, limit, offset int) (*dto.ProductListResponse, error) {
						return &dto.ProductListResponse{
							Items: []dto.ProductResponse{*validProductResponse()},
							Page:  dto.PageResponse{Limit: limit, Offset: offset},
						}, nil
					},
				}
			},
			companyID:      testCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.ProductListResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Len(t, out.Items, 1)
				assert.Equal(t, 20, out.Page.Limit)
				assert.Equal(t, 0, out.Page.Offset)
			},
		},
		{
			name:  "Success_WithPagination",
			query: "?limit=10&offset=5",
			mockSetup: func() *fakeProductUseCase {
				return &fakeProductUseCase{
					listFunc: func(_ string, limit, offset int) (*dto.ProductListResponse, error) {
						return &dto.ProductListResponse{
							Items: []dto.ProductResponse{},
							Page:  dto.PageResponse{Limit: limit, Offset: offset},
						}, nil
					},
				}
			},
			companyID:      testCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.ProductListResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, 10, out.Page.Limit)
				assert.Equal(t, 5, out.Page.Offset)
			},
		},
		{
			name:           "Unauthorized_NoCompanyID",
			query:          "",
			mockSetup:      func() *fakeProductUseCase { return &fakeProductUseCase{} },
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
			mockSetup: func() *fakeProductUseCase {
				return &fakeProductUseCase{
					listFunc: func(_ string, _, _ int) (*dto.ProductListResponse, error) {
						return nil, errors.New("db error")
					},
				}
			},
			companyID:      testCompanyID,
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

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			if tt.companyID != "" {
				app.Use(mockCompanyMiddleware(tt.companyID))
			} else {
				app.Use(func(c *fiber.Ctx) error { return c.Next() })
			}
			app.Get("/products", NewProductHandler(fakeUC).List)

			req := httptest.NewRequest(http.MethodGet, "/products"+tt.query, nil)

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

// ── Tests Update ───────────────────────────────────────────────────────────────

func TestProductHandler_Update(t *testing.T) {
	name := "Nombre actualizado"
	price := decimal.NewFromInt(15000)

	tests := []struct {
		name           string
		id             string
		body           interface{}
		mockSetup      func() *fakeProductUseCase
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			id:   "prod-123",
			body: dto.UpdateProductRequest{
				Name:  &name,
				Price: &price,
			},
			mockSetup: func() *fakeProductUseCase {
				return &fakeProductUseCase{
					updateFunc: func(id string, _ dto.UpdateProductRequest) (*dto.ProductResponse, error) {
						updated := validProductResponse()
						updated.Name = "Nombre actualizado"
						updated.Price = price
						return updated, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.ProductResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "Nombre actualizado", out.Name)
			},
		},
		{
			name:           "BadRequest_MissingID",
			id:             "",
			body:           dto.UpdateProductRequest{Name: &name},
			mockSetup:      func() *fakeProductUseCase { return &fakeProductUseCase{} },
			expectedStatus: http.StatusNotFound,
			validateBody:   nil,
		},
		{
			name:           "InvalidBody",
			id:             "prod-123",
			body:           "invalid json",
			mockSetup:      func() *fakeProductUseCase { return &fakeProductUseCase{} },
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INVALID_BODY", errResp.Code)
			},
		},
		{
			name: "NotFound",
			id:   "prod-999",
			body: dto.UpdateProductRequest{Name: &name},
			mockSetup: func() *fakeProductUseCase {
				return &fakeProductUseCase{
					updateFunc: func(_ string, _ dto.UpdateProductRequest) (*dto.ProductResponse, error) {
						return nil, nil
					},
				}
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_FOUND", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			id:   "prod-123",
			body: dto.UpdateProductRequest{Name: &name},
			mockSetup: func() *fakeProductUseCase {
				return &fakeProductUseCase{
					updateFunc: func(_ string, _ dto.UpdateProductRequest) (*dto.ProductResponse, error) {
						return nil, errors.New("db error")
					},
				}
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
			fakeUC := tt.mockSetup()

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockCompanyMiddleware(testCompanyID))
			app.Put("/products/:id", NewProductHandler(fakeUC).Update)

			path := "/products/"
			if tt.id != "" {
				path += tt.id
			}
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(bodyBytes))
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
