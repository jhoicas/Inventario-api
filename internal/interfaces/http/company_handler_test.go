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
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// ── Fake CompanyUseCase (mock manual para tests) ──────────────────────────────────

type fakeCompanyUseCase struct {
	createFunc  func(in dto.CreateCompanyRequest) (*dto.CompanyResponse, error)
	getByIDFunc func(id string) (*dto.CompanyResponse, error)
	listFunc    func(limit, offset int) (*dto.CompanyListResponse, error)
}

func (f *fakeCompanyUseCase) Create(in dto.CreateCompanyRequest) (*dto.CompanyResponse, error) {
	if f.createFunc != nil {
		return f.createFunc(in)
	}
	return nil, errors.New("create not configured")
}

func (f *fakeCompanyUseCase) GetByID(id string) (*dto.CompanyResponse, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(id)
	}
	return nil, errors.New("getByID not configured")
}

func (f *fakeCompanyUseCase) List(limit, offset int) (*dto.CompanyListResponse, error) {
	if f.listFunc != nil {
		return f.listFunc(limit, offset)
	}
	return nil, errors.New("list not configured")
}

// ── Helpers ────────────────────────────────────────────────────────────────────

const companyTestID = "company-123"

// Los endpoints de companies son públicos (onboarding); no requieren company_id en contexto.
// mockCompanyAuthMiddleware se deja por si en el futuro Get/Update requieren autenticación.
func mockCompanyAuthMiddleware(companyID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if companyID != "" {
			c.Locals(LocalCompanyID, companyID)
		}
		return c.Next()
	}
}

func validCreateCompanyRequest() dto.CreateCompanyRequest {
	return dto.CreateCompanyRequest{
		Name:    "Empresa Test S.A.S",
		NIT:     "900123456-1",
		Address: "Calle 100 #15-20",
		Phone:   "+57 300 123 4567",
		Email:   "contacto@empresatest.com",
	}
}

func validCompanyResponse() *dto.CompanyResponse {
	now := time.Now()
	return &dto.CompanyResponse{
		ID:        companyTestID,
		Name:      "Empresa Test S.A.S",
		NIT:       "900123456-1",
		Address:   "Calle 100 #15-20",
		Phone:     "+57 300 123 4567",
		Email:     "contacto@empresatest.com",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ── Tests Create ─────────────────────────────────────────────────────────────────

func TestCompanyHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func() *fakeCompanyUseCase
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			body: validCreateCompanyRequest(),
			mockSetup: func() *fakeCompanyUseCase {
				return &fakeCompanyUseCase{
					createFunc: func(in dto.CreateCompanyRequest) (*dto.CompanyResponse, error) {
						assert.Equal(t, "Empresa Test S.A.S", in.Name)
						assert.Equal(t, "900123456-1", in.NIT)
						return validCompanyResponse(), nil
					},
				}
			},
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.CompanyResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, companyTestID, out.ID)
				assert.Equal(t, "Empresa Test S.A.S", out.Name)
				assert.Equal(t, "900123456-1", out.NIT)
				assert.Equal(t, "active", out.Status)
			},
		},
		{
			name:           "InvalidBody",
			body:           "not valid json",
			mockSetup:      func() *fakeCompanyUseCase { return &fakeCompanyUseCase{} },
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INVALID_BODY", errResp.Code)
			},
		},
		{
			name: "Validation_NameAndNITRequired",
			body: dto.CreateCompanyRequest{Name: "", NIT: ""},
			mockSetup:      func() *fakeCompanyUseCase { return &fakeCompanyUseCase{} },
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "VALIDATION", errResp.Code)
				assert.Contains(t, errResp.Message, "name")
			},
		},
		{
			name: "Conflict_DuplicateNIT",
			body: validCreateCompanyRequest(),
			mockSetup: func() *fakeCompanyUseCase {
				return &fakeCompanyUseCase{
					createFunc: func(_ dto.CreateCompanyRequest) (*dto.CompanyResponse, error) {
						return nil, domain.ErrDuplicate
					},
				}
			},
			expectedStatus: http.StatusConflict,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "DUPLICATE", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			body: validCreateCompanyRequest(),
			mockSetup: func() *fakeCompanyUseCase {
				return &fakeCompanyUseCase{
					createFunc: func(_ dto.CreateCompanyRequest) (*dto.CompanyResponse, error) {
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
			handler := NewCompanyHandler(fakeUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Post("/companies", handler.Create)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewReader(bodyBytes))
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

func TestCompanyHandler_GetByID(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockSetup      func() *fakeCompanyUseCase
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			id:   companyTestID,
			mockSetup: func() *fakeCompanyUseCase {
				return &fakeCompanyUseCase{
					getByIDFunc: func(id string) (*dto.CompanyResponse, error) {
						return validCompanyResponse(), nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.CompanyResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, companyTestID, out.ID)
				assert.Equal(t, "Empresa Test S.A.S", out.Name)
			},
		},
		{
			name:           "BadRequest_MissingID",
			id:             "",
			mockSetup:      func() *fakeCompanyUseCase { return &fakeCompanyUseCase{} },
			// Enrutador devuelve 404 cuando la ruta no hace match (id ausente).
			expectedStatus: http.StatusNotFound,
			validateBody:   nil,
		},
		{
			name: "NotFound",
			id:   "company-999",
			mockSetup: func() *fakeCompanyUseCase {
				return &fakeCompanyUseCase{
					getByIDFunc: func(_ string) (*dto.CompanyResponse, error) {
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
			id:   companyTestID,
			mockSetup: func() *fakeCompanyUseCase {
				return &fakeCompanyUseCase{
					getByIDFunc: func(_ string) (*dto.CompanyResponse, error) {
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
			handler := NewCompanyHandler(fakeUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Get("/companies/:id", handler.GetByID)

			path := "/companies/"
			if tt.id != "" {
				path += tt.id
			} else {
				path = "/companies//" // id vacío: Params("id") = ""
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

func TestCompanyHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		mockSetup      func() *fakeCompanyUseCase
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name:  "Success_DefaultPagination",
			query: "",
			mockSetup: func() *fakeCompanyUseCase {
				return &fakeCompanyUseCase{
					listFunc: func(limit, offset int) (*dto.CompanyListResponse, error) {
						assert.Equal(t, 20, limit)
						assert.Equal(t, 0, offset)
						return &dto.CompanyListResponse{
							Items: []dto.CompanyResponse{*validCompanyResponse()},
							Page:  dto.PageResponse{Limit: 20, Offset: 0},
						}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.CompanyListResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Len(t, out.Items, 1)
				assert.Equal(t, companyTestID, out.Items[0].ID)
				assert.Equal(t, 20, out.Page.Limit)
				assert.Equal(t, 0, out.Page.Offset)
			},
		},
		{
			name:  "Success_WithPagination",
			query: "?limit=10&offset=5",
			mockSetup: func() *fakeCompanyUseCase {
				return &fakeCompanyUseCase{
					listFunc: func(limit, offset int) (*dto.CompanyListResponse, error) {
						assert.Equal(t, 10, limit)
						assert.Equal(t, 5, offset)
						return &dto.CompanyListResponse{
							Items: []dto.CompanyResponse{},
							Page:  dto.PageResponse{Limit: 10, Offset: 5},
						}, nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.CompanyListResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Len(t, out.Items, 0)
				assert.Equal(t, 10, out.Page.Limit)
				assert.Equal(t, 5, out.Page.Offset)
			},
		},
		{
			name:  "UseCase_InternalError",
			query: "",
			mockSetup: func() *fakeCompanyUseCase {
				return &fakeCompanyUseCase{
					listFunc: func(_, _ int) (*dto.CompanyListResponse, error) {
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
			handler := NewCompanyHandler(fakeUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Get("/companies", handler.List)

			req := httptest.NewRequest(http.MethodGet, "/companies"+tt.query, nil)

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
