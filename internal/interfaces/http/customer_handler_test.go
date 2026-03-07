package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// ── Fake CustomerUseCase (mock manual para tests) ──────────────────────────────

type fakeCustomerUseCase struct {
	createFunc func(companyID string, in dto.CreateCustomerRequest) (*dto.CustomerResponse, error)
	listFunc   func(companyID string, limit, offset int) ([]*dto.CustomerResponse, error)
}

func (f *fakeCustomerUseCase) Create(companyID string, in dto.CreateCustomerRequest) (*dto.CustomerResponse, error) {
	if f.createFunc != nil {
		return f.createFunc(companyID, in)
	}
	return nil, errors.New("create not configured")
}

func (f *fakeCustomerUseCase) List(companyID string, limit, offset int) ([]*dto.CustomerResponse, error) {
	if f.listFunc != nil {
		return f.listFunc(companyID, limit, offset)
	}
	return nil, errors.New("list not configured")
}

// ── Helpers ────────────────────────────────────────────────────────────────────

const customerTestCompanyID = "company-123"

// mockCustomerAuthMiddleware inyecta company_id en c.Locals para simular AuthMiddleware.
func mockCustomerAuthMiddleware(companyID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if companyID != "" {
			c.Locals(LocalCompanyID, companyID)
		}
		return c.Next()
	}
}

func validCreateCustomerRequest() dto.CreateCustomerRequest {
	return dto.CreateCustomerRequest{
		Name:  "Cliente Test",
		TaxID: "123456789-0",
		Email: "cliente@example.com",
		Phone: "+57 3000000000",
	}
}

func validCustomerResponse() *dto.CustomerResponse {
	return &dto.CustomerResponse{
		ID:        "cust-001",
		CompanyID: customerTestCompanyID,
		Name:      "Cliente Test",
		TaxID:     "123456789-0",
		Email:     "cliente@example.com",
		Phone:     "+57 3000000000",
	}
}

// ── Tests Create ────────────────────────────────────────────────────────────────

func TestCustomerHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func() *fakeCustomerUseCase
		companyID      string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			body: validCreateCustomerRequest(),
			mockSetup: func() *fakeCustomerUseCase {
				return &fakeCustomerUseCase{
					createFunc: func(_ string, _ dto.CreateCustomerRequest) (*dto.CustomerResponse, error) {
						return validCustomerResponse(), nil
					},
				}
			},
			companyID:      customerTestCompanyID,
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.CustomerResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "cust-001", out.ID)
				assert.Equal(t, "Cliente Test", out.Name)
				assert.Equal(t, "123456789-0", out.TaxID)
			},
		},
		{
			name:           "Unauthorized_NoCompanyID",
			body:           validCreateCustomerRequest(),
			mockSetup:      func() *fakeCustomerUseCase { return &fakeCustomerUseCase{} },
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
			mockSetup:      func() *fakeCustomerUseCase { return &fakeCustomerUseCase{} },
			companyID:      customerTestCompanyID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INVALID_BODY", errResp.Code)
			},
		},
		{
			name: "Validation_ErrInvalidInput",
			body: dto.CreateCustomerRequest{
				Name:  "",
				TaxID: "",
			},
			mockSetup: func() *fakeCustomerUseCase {
				return &fakeCustomerUseCase{
					createFunc: func(_ string, _ dto.CreateCustomerRequest) (*dto.CustomerResponse, error) {
						return nil, domain.ErrInvalidInput
					},
				}
			},
			companyID:      customerTestCompanyID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "VALIDATION", errResp.Code)
			},
		},
		{
			name: "Duplicate_TaxIDExists",
			body: validCreateCustomerRequest(),
			mockSetup: func() *fakeCustomerUseCase {
				return &fakeCustomerUseCase{
					createFunc: func(_ string, _ dto.CreateCustomerRequest) (*dto.CustomerResponse, error) {
						return nil, domain.ErrDuplicate
					},
				}
			},
			companyID:      customerTestCompanyID,
			expectedStatus: http.StatusConflict,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "DUPLICATE", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			body: validCreateCustomerRequest(),
			mockSetup: func() *fakeCustomerUseCase {
				return &fakeCustomerUseCase{
					createFunc: func(_ string, _ dto.CreateCustomerRequest) (*dto.CustomerResponse, error) {
						return nil, errors.New("db error")
					},
				}
			},
			companyID:      customerTestCompanyID,
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
			handler := NewCustomerHandler(fakeUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			if tt.companyID != "" {
				app.Use(mockCustomerAuthMiddleware(tt.companyID))
			} else {
				app.Use(func(c *fiber.Ctx) error { return c.Next() })
			}
			app.Post("/customers", handler.Create)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewReader(bodyBytes))
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

// ── Tests List ──────────────────────────────────────────────────────────────────

func TestCustomerHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		mockSetup      func() *fakeCustomerUseCase
		companyID      string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name:  "Success_DefaultPagination",
			query: "",
			mockSetup: func() *fakeCustomerUseCase {
				return &fakeCustomerUseCase{
					listFunc: func(companyID string, limit, offset int) ([]*dto.CustomerResponse, error) {
						assert.Equal(t, customerTestCompanyID, companyID)
						assert.Equal(t, 20, limit)
						assert.Equal(t, 0, offset)
						return []*dto.CustomerResponse{validCustomerResponse()}, nil
					},
				}
			},
			companyID:      customerTestCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out []*dto.CustomerResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Len(t, out, 1)
				assert.Equal(t, "cust-001", out[0].ID)
			},
		},
		{
			name:  "Success_WithPagination",
			query: "?limit=10&offset=5",
			mockSetup: func() *fakeCustomerUseCase {
				return &fakeCustomerUseCase{
					listFunc: func(companyID string, limit, offset int) ([]*dto.CustomerResponse, error) {
						assert.Equal(t, customerTestCompanyID, companyID)
						assert.Equal(t, 10, limit)
						assert.Equal(t, 5, offset)
						return []*dto.CustomerResponse{}, nil
					},
				}
			},
			companyID:      customerTestCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out []*dto.CustomerResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Len(t, out, 0)
			},
		},
		{
			name:           "Unauthorized_NoCompanyID",
			query:          "",
			mockSetup:      func() *fakeCustomerUseCase { return &fakeCustomerUseCase{} },
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
			mockSetup: func() *fakeCustomerUseCase {
				return &fakeCustomerUseCase{
					listFunc: func(_ string, _, _ int) ([]*dto.CustomerResponse, error) {
						return nil, errors.New("db error")
					},
				}
			},
			companyID:      customerTestCompanyID,
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
			handler := NewCustomerHandler(fakeUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			if tt.companyID != "" {
				app.Use(mockCustomerAuthMiddleware(tt.companyID))
			} else {
				app.Use(func(c *fiber.Ctx) error { return c.Next() })
			}
			app.Get("/customers", handler.List)

			req := httptest.NewRequest(http.MethodGet, "/customers"+tt.query, nil)

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

