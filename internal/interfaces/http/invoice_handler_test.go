package http

import (
	"bytes"
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
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// ── Fakes para los UseCases ───────────────────────────────────────────────────

type fakeCreateInvoiceUseCase struct {
	createInvoiceFunc        func(ctx context.Context, companyID, userID string, in dto.CreateInvoiceRequest) (*dto.InvoiceResponse, error)
	getInvoiceDIANStatusFunc func(ctx context.Context, companyID, id string) (*dto.InvoiceDIANStatusDTO, error)
	getInvoiceFunc           func(ctx context.Context, companyID, id string) (*dto.InvoiceResponse, error)
}

func (f *fakeCreateInvoiceUseCase) CreateInvoice(ctx context.Context, companyID, userID string, in dto.CreateInvoiceRequest) (*dto.InvoiceResponse, error) {
	if f.createInvoiceFunc != nil {
		return f.createInvoiceFunc(ctx, companyID, userID, in)
	}
	return nil, errors.New("createInvoice not configured")
}

func (f *fakeCreateInvoiceUseCase) GetInvoiceDIANStatus(ctx context.Context, companyID, id string) (*dto.InvoiceDIANStatusDTO, error) {
	if f.getInvoiceDIANStatusFunc != nil {
		return f.getInvoiceDIANStatusFunc(ctx, companyID, id)
	}
	return nil, errors.New("getInvoiceDIANStatus not configured")
}

func (f *fakeCreateInvoiceUseCase) GetInvoice(ctx context.Context, companyID, id string) (*dto.InvoiceResponse, error) {
	if f.getInvoiceFunc != nil {
		return f.getInvoiceFunc(ctx, companyID, id)
	}
	return nil, errors.New("getInvoice not configured")
}

type fakeCreateCreditNoteUseCase struct {
	createCreditNoteFunc func(ctx context.Context, companyID, userID, invoiceID string, in dto.ReturnInvoiceRequest) (*dto.InvoiceResponse, error)
}

func (f *fakeCreateCreditNoteUseCase) CreateCreditNote(ctx context.Context, companyID, userID, invoiceID string, in dto.ReturnInvoiceRequest) (*dto.InvoiceResponse, error) {
	if f.createCreditNoteFunc != nil {
		return f.createCreditNoteFunc(ctx, companyID, userID, invoiceID, in)
	}
	return nil, errors.New("createCreditNote not configured")
}

type fakeCreateDebitNoteUseCase struct {
	createDebitNoteFunc func(ctx context.Context, companyID, userID, invoiceID string, in dto.CreateDebitNoteRequest) (*dto.DebitNoteResponse, error)
}

func (f *fakeCreateDebitNoteUseCase) CreateDebitNote(ctx context.Context, companyID, userID, invoiceID string, in dto.CreateDebitNoteRequest) (*dto.DebitNoteResponse, error) {
	if f.createDebitNoteFunc != nil {
		return f.createDebitNoteFunc(ctx, companyID, userID, invoiceID, in)
	}
	return nil, errors.New("createDebitNote not configured")
}

type fakeVoidInvoiceUseCase struct {
	voidInvoiceFunc func(ctx context.Context, companyID, userID, invoiceID string, in dto.CreateVoidInvoiceRequest) (*dto.VoidInvoiceResponse, error)
}

func (f *fakeVoidInvoiceUseCase) VoidInvoice(ctx context.Context, companyID, userID, invoiceID string, in dto.CreateVoidInvoiceRequest) (*dto.VoidInvoiceResponse, error) {
	if f.voidInvoiceFunc != nil {
		return f.voidInvoiceFunc(ctx, companyID, userID, invoiceID, in)
	}
	return nil, errors.New("voidInvoice not configured")
}

type fakeInvoicePDFUseCase struct {
	downloadInvoicePDFFunc func(ctx context.Context, companyID, invoiceID string) ([]byte, string, error)
}

func (f *fakeInvoicePDFUseCase) DownloadInvoicePDF(ctx context.Context, companyID, invoiceID string) ([]byte, string, error) {
	if f.downloadInvoicePDFFunc != nil {
		return f.downloadInvoicePDFFunc(ctx, companyID, invoiceID)
	}
	return nil, "", errors.New("downloadInvoicePDF not configured")
}

// ── Helpers ────────────────────────────────────────────────────────────────────

const invoiceTestCompanyID = "company-123"
const invoiceTestUserID = "user-456"

func mockInvoiceAuthMiddleware(companyID, userID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(LocalCompanyID, companyID)
		c.Locals(LocalUserID, userID)
		return c.Next()
	}
}

func mockInvoiceAuthCompanyOnly(companyID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(LocalCompanyID, companyID)
		c.Locals(LocalUserID, invoiceTestUserID)
		return c.Next()
	}
}

func validCreateInvoiceRequest() dto.CreateInvoiceRequest {
	return dto.CreateInvoiceRequest{
		CustomerID:  "cust-001",
		WarehouseID: "wh-001",
		Prefix:      "FV",
		Items: []dto.InvoiceItemRequest{
			{ProductID: "prod-001", Quantity: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(10000)},
		},
	}
}

func validReturnInvoiceRequest() dto.ReturnInvoiceRequest {
	return dto.ReturnInvoiceRequest{
		WarehouseID: "wh-001",
		Items: []dto.ReturnItemRequest{
			{ProductID: "prod-001", Quantity: decimal.NewFromInt(1)},
		},
		Reason: "Devolución de prueba",
	}
}

func validCreateDebitNoteRequest() dto.CreateDebitNoteRequest {
	return dto.CreateDebitNoteRequest{
		Reason: "Ajuste por mayor valor",
		Items: []dto.DebitNoteItemRequest{
			{ProductID: "prod-001", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(5000)},
		},
	}
}

func validInvoiceResponse() *dto.InvoiceResponse {
	return &dto.InvoiceResponse{
		ID:           "inv-123",
		CompanyID:    invoiceTestCompanyID,
		CustomerID:   "cust-001",
		CustomerName: "Cliente Test",
		Prefix:       "FV",
		Number:       "00001",
		Date:         "2025-01-15",
		NetTotal:     decimal.NewFromInt(20000),
		TaxTotal:     decimal.NewFromInt(3800),
		GrandTotal:   decimal.NewFromInt(23800),
		DIAN_Status:  "EXITOSO",
		CUFE:         "abc123",
		Details:      []dto.InvoiceDetailResponse{},
	}
}

func validInvoiceDIANStatusDTO() *dto.InvoiceDIANStatusDTO {
	return &dto.InvoiceDIANStatusDTO{
		ID:         "inv-123",
		DIANStatus: "EXITOSO",
		CUFE:       "abc123",
		TrackID:    "track-001",
		Errors:     "",
	}
}

func validDebitNoteResponse() *dto.DebitNoteResponse {
	return &dto.DebitNoteResponse{
		DebitNoteID: "dn-123",
		CUFE:        "cude-123",
		DIANStatus:  "EXITOSO",
	}
}

func validVoidInvoiceRequest() dto.CreateVoidInvoiceRequest {
	return dto.CreateVoidInvoiceRequest{
		ConceptCode: 2,
		Reason:      "Anulación de prueba",
	}
}

func validVoidInvoiceResponse() *dto.VoidInvoiceResponse {
	return &dto.VoidInvoiceResponse{
		CreditNoteID: "cn-void-123",
		CUFE:         "cufe-void-123",
		DIANStatus:   "EXITOSO",
	}
}

// ── Tests Create ────────────────────────────────────────────────────────────────

func TestInvoiceHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase)
		companyID      string
		userID         string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			body: validCreateInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					createInvoiceFunc: func(_ context.Context, _, _ string, _ dto.CreateInvoiceRequest) (*dto.InvoiceResponse, error) {
						return validInvoiceResponse(), nil
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.InvoiceResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "inv-123", out.ID)
				assert.Equal(t, "FV00001", out.Prefix+out.Number)
			},
		},
		{
			name: "Unauthorized_NoCompanyID",
			body: validCreateInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      "",
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "UNAUTHORIZED", errResp.Code)
			},
		},
		{
			name: "Unauthorized_NoUserID",
			body: validCreateInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
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
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INVALID_BODY", errResp.Code)
			},
		},
		{
			name: "Validation_ErrInvalidInput",
			body: validCreateInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					createInvoiceFunc: func(_ context.Context, _, _ string, _ dto.CreateInvoiceRequest) (*dto.InvoiceResponse, error) {
						return nil, domain.ErrInvalidInput
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "VALIDATION", errResp.Code)
			},
		},
		{
			name: "NotFound",
			body: validCreateInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					createInvoiceFunc: func(_ context.Context, _, _ string, _ dto.CreateInvoiceRequest) (*dto.InvoiceResponse, error) {
						return nil, domain.ErrNotFound
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_FOUND", errResp.Code)
			},
		},
		{
			name: "Forbidden",
			body: validCreateInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					createInvoiceFunc: func(_ context.Context, _, _ string, _ dto.CreateInvoiceRequest) (*dto.InvoiceResponse, error) {
						return nil, domain.ErrForbidden
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusForbidden,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "FORBIDDEN", errResp.Code)
			},
		},
		{
			name: "InsufficientStock",
			body: validCreateInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					createInvoiceFunc: func(_ context.Context, _, _ string, _ dto.CreateInvoiceRequest) (*dto.InvoiceResponse, error) {
						return nil, domain.ErrInsufficientStock
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusConflict,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INSUFFICIENT_STOCK", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			body: validCreateInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					createInvoiceFunc: func(_ context.Context, _, _ string, _ dto.CreateInvoiceRequest) (*dto.InvoiceResponse, error) {
						return nil, errors.New("db connection failed")
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
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
			createUC, returnUC, pdfUC := tt.mockSetup()
			handler := NewInvoiceHandler(createUC, returnUC, pdfUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockInvoiceAuthMiddleware(tt.companyID, tt.userID))
			app.Post("/invoices", handler.Create)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewReader(bodyBytes))
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

// ── Tests HandleReturn ──────────────────────────────────────────────────────────

func TestInvoiceHandler_HandleReturn(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		body           interface{}
		mockSetup      func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase)
		companyID      string
		userID         string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			id:   "inv-123",
			body: validReturnInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				returnUC := &fakeCreateCreditNoteUseCase{
					createCreditNoteFunc: func(_ context.Context, _, _, _ string, _ dto.ReturnInvoiceRequest) (*dto.InvoiceResponse, error) {
						return validInvoiceResponse(), nil
					},
				}
				return &fakeCreateInvoiceUseCase{}, returnUC, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.InvoiceResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "inv-123", out.ID)
			},
		},
		{
			name: "Unauthorized_NoCompanyID",
			id:   "inv-123",
			body: validReturnInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      "",
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "UNAUTHORIZED", errResp.Code)
			},
		},
		{
			name: "BadRequest_MissingID",
			id:   "",
			body: validReturnInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID: invoiceTestCompanyID,
			userID:    invoiceTestUserID,
			// La ruta no hace match cuando falta :id, Fiber responde 404.
			expectedStatus: http.StatusNotFound,
			validateBody:   nil,
		},
		{
			name: "InvalidBody",
			id:   "inv-123",
			body: "invalid json",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INVALID_BODY", errResp.Code)
			},
		},
		{
			name: "NotFound",
			id:   "inv-999",
			body: validReturnInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				returnUC := &fakeCreateCreditNoteUseCase{
					createCreditNoteFunc: func(_ context.Context, _, _, _ string, _ dto.ReturnInvoiceRequest) (*dto.InvoiceResponse, error) {
						return nil, domain.ErrNotFound
					},
				}
				return &fakeCreateInvoiceUseCase{}, returnUC, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_FOUND", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			id:   "inv-123",
			body: validReturnInvoiceRequest(),
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				returnUC := &fakeCreateCreditNoteUseCase{
					createCreditNoteFunc: func(_ context.Context, _, _, _ string, _ dto.ReturnInvoiceRequest) (*dto.InvoiceResponse, error) {
						return nil, errors.New("db error")
					},
				}
				return &fakeCreateInvoiceUseCase{}, returnUC, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
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
			createUC, returnUC, pdfUC := tt.mockSetup()
			handler := NewInvoiceHandler(createUC, returnUC, pdfUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockInvoiceAuthMiddleware(tt.companyID, tt.userID))
			app.Post("/invoices/:id/return", handler.HandleReturn)

			path := "/invoices/"
			if tt.id != "" {
				path += tt.id + "/return"
			} else {
				path = "/invoices//return" // id vacío: Params("id") = ""
			}
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(bodyBytes))
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

// ── Tests HandleDebitNote ──────────────────────────────────────────────────────

func TestInvoiceHandler_HandleDebitNote(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		body           interface{}
		debitUC        *fakeCreateDebitNoteUseCase
		companyID      string
		userID         string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			id:   "inv-123",
			body: validCreateDebitNoteRequest(),
			debitUC: &fakeCreateDebitNoteUseCase{
				createDebitNoteFunc: func(_ context.Context, _, _, _ string, _ dto.CreateDebitNoteRequest) (*dto.DebitNoteResponse, error) {
					return validDebitNoteResponse(), nil
				},
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.DebitNoteResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "dn-123", out.DebitNoteID)
				assert.Equal(t, "EXITOSO", out.DIANStatus)
			},
		},
		{
			name:           "Unauthorized_NoCompanyID",
			id:             "inv-123",
			body:           validCreateDebitNoteRequest(),
			debitUC:        &fakeCreateDebitNoteUseCase{},
			companyID:      "",
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "UNAUTHORIZED", errResp.Code)
			},
		},
		{
			name:           "InvalidBody",
			id:             "inv-123",
			body:           "invalid json",
			debitUC:        &fakeCreateDebitNoteUseCase{},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INVALID_BODY", errResp.Code)
			},
		},
		{
			name: "NotFound",
			id:   "inv-999",
			body: validCreateDebitNoteRequest(),
			debitUC: &fakeCreateDebitNoteUseCase{
				createDebitNoteFunc: func(_ context.Context, _, _, _ string, _ dto.CreateDebitNoteRequest) (*dto.DebitNoteResponse, error) {
					return nil, domain.ErrNotFound
				},
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_FOUND", errResp.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewInvoiceHandlerWithDebit(
				&fakeCreateInvoiceUseCase{},
				&fakeCreateCreditNoteUseCase{},
				tt.debitUC,
				&fakeInvoicePDFUseCase{},
			)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockInvoiceAuthMiddleware(tt.companyID, tt.userID))
			app.Post("/invoices/:id/debit-note", handler.HandleDebitNote)

			path := "/invoices/"
			if tt.id != "" {
				path += tt.id + "/debit-note"
			} else {
				path = "/invoices//debit-note"
			}

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(bodyBytes))
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

	t.Run("ServiceUnavailable_NilDebitUC", func(t *testing.T) {
		handler := NewInvoiceHandler(
			&fakeCreateInvoiceUseCase{},
			&fakeCreateCreditNoteUseCase{},
			&fakeInvoicePDFUseCase{},
		)

		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		app.Use(mockInvoiceAuthMiddleware(invoiceTestCompanyID, invoiceTestUserID))
		app.Post("/invoices/:id/debit-note", handler.HandleDebitNote)

		bodyBytes, _ := json.Marshal(validCreateDebitNoteRequest())
		req := httptest.NewRequest(http.MethodPost, "/invoices/inv-123/debit-note", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		var errResp dto.ErrorResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
		assert.Equal(t, "INTERNAL", errResp.Code)
	})
}

// ── Tests HandleVoidInvoice ────────────────────────────────────────────────────

func TestInvoiceHandler_HandleVoidInvoice(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		body           interface{}
		voidUC         *fakeVoidInvoiceUseCase
		companyID      string
		userID         string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			id:   "inv-123",
			body: validVoidInvoiceRequest(),
			voidUC: &fakeVoidInvoiceUseCase{
				voidInvoiceFunc: func(_ context.Context, _, _, _ string, _ dto.CreateVoidInvoiceRequest) (*dto.VoidInvoiceResponse, error) {
					return validVoidInvoiceResponse(), nil
				},
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.VoidInvoiceResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "cn-void-123", out.CreditNoteID)
				assert.Equal(t, "EXITOSO", out.DIANStatus)
			},
		},
		{
			name:           "Unauthorized_NoCompanyID",
			id:             "inv-123",
			body:           validVoidInvoiceRequest(),
			voidUC:         &fakeVoidInvoiceUseCase{},
			companyID:      "",
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "UNAUTHORIZED", errResp.Code)
			},
		},
		{
			name:           "InvalidBody",
			id:             "inv-123",
			body:           "invalid json",
			voidUC:         &fakeVoidInvoiceUseCase{},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INVALID_BODY", errResp.Code)
			},
		},
		{
			name: "NotFound",
			id:   "inv-999",
			body: validVoidInvoiceRequest(),
			voidUC: &fakeVoidInvoiceUseCase{
				voidInvoiceFunc: func(_ context.Context, _, _, _ string, _ dto.CreateVoidInvoiceRequest) (*dto.VoidInvoiceResponse, error) {
					return nil, domain.ErrNotFound
				},
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_FOUND", errResp.Code)
			},
		},
		{
			name: "Conflict_NotSent",
			id:   "inv-123",
			body: validVoidInvoiceRequest(),
			voidUC: &fakeVoidInvoiceUseCase{
				voidInvoiceFunc: func(_ context.Context, _, _, _ string, _ dto.CreateVoidInvoiceRequest) (*dto.VoidInvoiceResponse, error) {
					return nil, domain.ErrConflict
				},
			},
			companyID:      invoiceTestCompanyID,
			userID:         invoiceTestUserID,
			expectedStatus: http.StatusConflict,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "CONFLICT", errResp.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewInvoiceHandlerWithBillingOps(
				&fakeCreateInvoiceUseCase{},
				&fakeCreateCreditNoteUseCase{},
				&fakeCreateDebitNoteUseCase{},
				tt.voidUC,
				&fakeInvoicePDFUseCase{},
			)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockInvoiceAuthMiddleware(tt.companyID, tt.userID))
			app.Post("/invoices/:id/void", handler.HandleVoidInvoice)

			path := "/invoices/"
			if tt.id != "" {
				path += tt.id + "/void"
			} else {
				path = "/invoices//void"
			}

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(bodyBytes))
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

	t.Run("ServiceUnavailable_NilVoidUC", func(t *testing.T) {
		handler := NewInvoiceHandlerWithBillingOps(
			&fakeCreateInvoiceUseCase{},
			&fakeCreateCreditNoteUseCase{},
			&fakeCreateDebitNoteUseCase{},
			nil,
			&fakeInvoicePDFUseCase{},
		)

		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		app.Use(mockInvoiceAuthMiddleware(invoiceTestCompanyID, invoiceTestUserID))
		app.Post("/invoices/:id/void", handler.HandleVoidInvoice)

		bodyBytes, _ := json.Marshal(validVoidInvoiceRequest())
		req := httptest.NewRequest(http.MethodPost, "/invoices/inv-123/void", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		var errResp dto.ErrorResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
		assert.Equal(t, "INTERNAL", errResp.Code)
	})
}

// ── Tests GetByID ───────────────────────────────────────────────────────────────

func TestInvoiceHandler_GetByID(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockSetup      func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase)
		companyID      string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			id:   "inv-123",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					getInvoiceFunc: func(_ context.Context, _, id string) (*dto.InvoiceResponse, error) {
						return validInvoiceResponse(), nil
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.InvoiceResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "inv-123", out.ID)
			},
		},
		{
			name: "Unauthorized_NoCompanyID",
			id:   "inv-123",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
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
			name: "BadRequest_MissingID",
			id:   "",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			expectedStatus: http.StatusNotFound,
			validateBody:   nil,
		},
		{
			name: "NotFound",
			id:   "inv-999",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					getInvoiceFunc: func(_ context.Context, _, _ string) (*dto.InvoiceResponse, error) {
						return nil, domain.ErrNotFound
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_FOUND", errResp.Code)
			},
		},
		{
			name: "Forbidden",
			id:   "inv-123",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					getInvoiceFunc: func(_ context.Context, _, _ string) (*dto.InvoiceResponse, error) {
						return nil, domain.ErrForbidden
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			expectedStatus: http.StatusForbidden,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "FORBIDDEN", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			id:   "inv-123",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					getInvoiceFunc: func(_ context.Context, _, _ string) (*dto.InvoiceResponse, error) {
						return nil, errors.New("db error")
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
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
			createUC, returnUC, pdfUC := tt.mockSetup()
			handler := NewInvoiceHandler(createUC, returnUC, pdfUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockInvoiceAuthCompanyOnly(tt.companyID))
			app.Get("/invoices/:id", handler.GetByID)

			path := "/invoices/"
			if tt.id != "" {
				path += tt.id
			} else {
				path = "/invoices//" // id vacío: Params("id") = ""
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

// ── Tests GetDIANStatus ─────────────────────────────────────────────────────────

func TestInvoiceHandler_GetDIANStatus(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockSetup      func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase)
		companyID      string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			id:   "inv-123",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					getInvoiceDIANStatusFunc: func(_ context.Context, _, id string) (*dto.InvoiceDIANStatusDTO, error) {
						return validInvoiceDIANStatusDTO(), nil
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.InvoiceDIANStatusDTO
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "inv-123", out.ID)
				assert.Equal(t, "EXITOSO", out.DIANStatus)
			},
		},
		{
			name: "Unauthorized_NoCompanyID",
			id:   "inv-123",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
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
			name: "BadRequest_MissingID",
			id:   "",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			expectedStatus: http.StatusNotFound,
			validateBody:   nil,
		},
		{
			name: "NotFound",
			id:   "inv-999",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					getInvoiceDIANStatusFunc: func(_ context.Context, _, _ string) (*dto.InvoiceDIANStatusDTO, error) {
						return nil, domain.ErrNotFound
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_FOUND", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			id:   "inv-123",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				uc := &fakeCreateInvoiceUseCase{
					getInvoiceDIANStatusFunc: func(_ context.Context, _, _ string) (*dto.InvoiceDIANStatusDTO, error) {
						return nil, errors.New("db error")
					},
				}
				return uc, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
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
			createUC, returnUC, pdfUC := tt.mockSetup()
			handler := NewInvoiceHandler(createUC, returnUC, pdfUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockInvoiceAuthCompanyOnly(tt.companyID))
			app.Get("/invoices/:id/status", handler.GetDIANStatus)

			path := "/invoices/"
			if tt.id != "" {
				path += tt.id + "/status"
			} else {
				path = "/invoices//status" // id vacío: Params("id") = ""
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

// ── Tests DownloadPDF ──────────────────────────────────────────────────────────

func TestInvoiceHandler_DownloadPDF(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockSetup      func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase)
		companyID      string
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			id:   "inv-123",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				pdfUC := &fakeInvoicePDFUseCase{
					downloadInvoicePDFFunc: func(_ context.Context, _, id string) ([]byte, string, error) {
						return []byte("%PDF-1.4 fake content"), "factura-inv-123.pdf", nil
					},
				}
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, pdfUC
			},
			companyID:      invoiceTestCompanyID,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
				assert.Contains(t, resp.Header.Get("Content-Disposition"), "factura-inv-123.pdf")
				assert.Equal(t, "21", resp.Header.Get("Content-Length"))
			},
		},
		{
			name: "Unauthorized_NoCompanyID",
			id:   "inv-123",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
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
			name: "BadRequest_MissingID",
			id:   "",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, &fakeInvoicePDFUseCase{}
			},
			companyID:      invoiceTestCompanyID,
			expectedStatus: http.StatusNotFound,
			validateBody:   nil,
		},
		{
			name: "NotFound",
			id:   "inv-999",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				pdfUC := &fakeInvoicePDFUseCase{
					downloadInvoicePDFFunc: func(_ context.Context, _, _ string) ([]byte, string, error) {
						return nil, "", domain.ErrNotFound
					},
				}
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, pdfUC
			},
			companyID:      invoiceTestCompanyID,
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_FOUND", errResp.Code)
			},
		},
		{
			name: "Forbidden",
			id:   "inv-123",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				pdfUC := &fakeInvoicePDFUseCase{
					downloadInvoicePDFFunc: func(_ context.Context, _, _ string) ([]byte, string, error) {
						return nil, "", domain.ErrForbidden
					},
				}
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, pdfUC
			},
			companyID:      invoiceTestCompanyID,
			expectedStatus: http.StatusForbidden,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "FORBIDDEN", errResp.Code)
			},
		},
		{
			name: "NotReady_ErrInvalidInput",
			id:   "inv-123",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				pdfUC := &fakeInvoicePDFUseCase{
					downloadInvoicePDFFunc: func(_ context.Context, _, _ string) ([]byte, string, error) {
						return nil, "", domain.ErrInvalidInput
					},
				}
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, pdfUC
			},
			companyID:      invoiceTestCompanyID,
			expectedStatus: http.StatusConflict,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "NOT_READY", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			id:   "inv-123",
			mockSetup: func() (*fakeCreateInvoiceUseCase, *fakeCreateCreditNoteUseCase, *fakeInvoicePDFUseCase) {
				pdfUC := &fakeInvoicePDFUseCase{
					downloadInvoicePDFFunc: func(_ context.Context, _, _ string) ([]byte, string, error) {
						return nil, "", errors.New("pdf generation failed")
					},
				}
				return &fakeCreateInvoiceUseCase{}, &fakeCreateCreditNoteUseCase{}, pdfUC
			},
			companyID:      invoiceTestCompanyID,
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
			createUC, returnUC, pdfUC := tt.mockSetup()
			handler := NewInvoiceHandler(createUC, returnUC, pdfUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mockInvoiceAuthCompanyOnly(tt.companyID))
			app.Get("/invoices/:id/pdf", handler.DownloadPDF)

			path := "/invoices/"
			if tt.id != "" {
				path += tt.id + "/pdf"
			} else {
				path = "/invoices//pdf" // id vacío: Params("id") = ""
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
