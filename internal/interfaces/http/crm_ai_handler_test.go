package http

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func mockCRMAIAuthMiddleware(companyID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if companyID != "" {
			c.Locals(LocalCompanyID, companyID)
		}
		return c.Next()
	}
}

func newCRMAITestApp(h *CRMAIHandler, companyID string) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(mockCRMAIAuthMiddleware(companyID))
	app.Post("/api/crm/ai/ask", h.AskAI)
	app.Post("/api/crm/sales/import", h.ImportSalesFile)
	return app
}

func TestAskAI_Success(t *testing.T) {
	// Arrange
	const companyID = "company-test-123"
	aiMock := new(mockAIAnalystService)
	bulkMock := new(mockBulkImporterService)
	log := logger.New(logger.Config{Env: "test", Level: "error"})
	handler := NewCRMAIHandler(aiMock, bulkMock, log)
	app := newCRMAITestApp(handler, companyID)

	expected := &dto.CRMTextToSQLResponse{
		Answer: "Resumen de prueba.",
		Data: []map[string]interface{}{
			{"producto": "A", "cantidad": 10.0},
			{"producto": "B", "cantidad": 5.0},
		},
		SQL: "SELECT 1",
	}
	aiMock.On("Ask", mock.Anything, companyID, "ventas").Return(expected, nil).Once()

	body := []byte(`{"question":"ventas"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/crm/ai/ask", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var out struct {
		Answer string                   `json:"answer"`
		Data   []map[string]interface{} `json:"data"`
		SQL    string                   `json:"sql"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	assert.Equal(t, "Resumen de prueba.", out.Answer)
	assert.Equal(t, "SELECT 1", out.SQL)
	assert.Len(t, out.Data, 2)
	assert.Equal(t, "A", out.Data[0]["producto"])
	aiMock.AssertExpectations(t)
}

func TestAskAI_InvalidBody(t *testing.T) {
	// Arrange
	const companyID = "company-test-123"
	aiMock := new(mockAIAnalystService)
	bulkMock := new(mockBulkImporterService)
	log := logger.New(logger.Config{Env: "test", Level: "error"})
	handler := NewCRMAIHandler(aiMock, bulkMock, log)
	app := newCRMAITestApp(handler, companyID)

	invalidJSON := []byte(`{"question":`) // JSON roto
	req := httptest.NewRequest(http.MethodPost, "/api/crm/ai/ask", bytes.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	aiMock.AssertNotCalled(t, "Ask", mock.Anything, mock.Anything, mock.Anything)
}

func TestImportSalesFile_Success(t *testing.T) {
	// Arrange
	const companyID = "company-test-123"
	aiMock := new(mockAIAnalystService)
	bulkMock := new(mockBulkImporterService)
	log := logger.New(logger.Config{Env: "test", Level: "error"})
	handler := NewCRMAIHandler(aiMock, bulkMock, log)
	app := newCRMAITestApp(handler, companyID)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "sales.csv")
	require.NoError(t, err)
	_, err = part.Write([]byte("customer_email,total_amount\nfoo@bar.com,100\n"))
	require.NoError(t, err)

	require.NoError(t, writer.WriteField("mappings", `{"format":"csv","table_name":"sales"}`))
	require.NoError(t, writer.Close())

	bulkMock.
		On("ImportFromCSV", mock.Anything, companyID, mock.AnythingOfType("string"), "sales").
		Return(1, nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/crm/sales/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Act
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var out map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	assert.Equal(t, "ok", out["status"])
	assert.Equal(t, float64(1), out["processed_records"])
	bulkMock.AssertExpectations(t)
}

func TestImportSalesFile_Success_WithColumnMappingsUTF8(t *testing.T) {
	// Arrange
	const companyID = "company-test-123"
	aiMock := new(mockAIAnalystService)
	bulkMock := new(mockBulkImporterService)
	log := logger.New(logger.Config{Env: "test", Level: "error"})
	handler := NewCRMAIHandler(aiMock, bulkMock, log)
	app := newCRMAITestApp(handler, companyID)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "ventas.csv")
	require.NoError(t, err)
	_, err = part.Write([]byte("Email,Fecha,Producto,Categoria\nfoo@bar.com,2026-04-01,Café Premium,Categoría\n"))
	require.NoError(t, err)

	columnMappings := `[{"sourceIndex":0,"sourceHeader":"Email","targetField":"correo"},{"sourceIndex":4,"sourceHeader":"Categoría","targetField":"categoria"}]`
	require.NoError(t, writer.WriteField("columnMappings", columnMappings))
	require.NoError(t, writer.Close())

	bulkMock.
		On("ImportFromCSV", mock.Anything, companyID, mock.AnythingOfType("string"), "sales").
		Return(1, nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/crm/sales/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Act
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	bulkMock.AssertExpectations(t)
}
