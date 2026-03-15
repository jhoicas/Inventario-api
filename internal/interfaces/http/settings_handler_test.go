package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDIANSettingsUseCase struct {
	saveFunc func(companyID string, in dto.UpsertDIANSettingsRequest) (*dto.DIANSettingsResponse, error)
}

func (f *fakeDIANSettingsUseCase) Save(companyID string, in dto.UpsertDIANSettingsRequest) (*dto.DIANSettingsResponse, error) {
	if f.saveFunc != nil {
		return f.saveFunc(companyID, in)
	}
	return nil, errors.New("save not configured")
}

func multipartRequest(t *testing.T, url, environment, password, filename string, fileData []byte) (*http.Request, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if environment != "" {
		require.NoError(t, writer.WriteField("environment", environment))
	}
	if password != "" {
		require.NoError(t, writer.WriteField("certificate_password", password))
	}
	if filename != "" {
		part, err := writer.CreateFormFile("certificate", filename)
		require.NoError(t, err)
		_, err = part.Write(fileData)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPut, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, writer.FormDataContentType()
}

func TestSettingsHandler_UpdateDIANSettings(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		fakeUC := &fakeDIANSettingsUseCase{saveFunc: func(companyID string, in dto.UpsertDIANSettingsRequest) (*dto.DIANSettingsResponse, error) {
			assert.Equal(t, "company-1", companyID)
			assert.Equal(t, "test", in.Environment)
			assert.Equal(t, "certificado.p12", in.CertificateFileName)
			assert.Equal(t, "secret123", in.CertificatePassword)
			assert.NotEmpty(t, in.CertificateData)
			return &dto.DIANSettingsResponse{
				CompanyID:           companyID,
				Environment:         in.Environment,
				CertificateFileName: "cert_20250101.p12",
				CertificateFileSize: int64(len(in.CertificateData)),
				UpdatedAt:           time.Now(),
			}, nil
		}}

		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		handler := NewSettingsHandler(fakeUC)
		app.Put("/api/settings/dian", func(c *fiber.Ctx) error {
			c.Locals(LocalCompanyID, "company-1")
			return c.Next()
		}, handler.UpdateDIANSettings)

		req, _ := multipartRequest(t, "/api/settings/dian", "test", "secret123", "certificado.p12", []byte("dummy-p12"))
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var out dto.DIANSettingsResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
		assert.Equal(t, "company-1", out.CompanyID)
		assert.Equal(t, "test", out.Environment)
	})

	t.Run("UnsupportedContentType", func(t *testing.T) {
		fakeUC := &fakeDIANSettingsUseCase{}
		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		handler := NewSettingsHandler(fakeUC)
		app.Put("/api/settings/dian", handler.UpdateDIANSettings)

		req := httptest.NewRequest(http.MethodPut, "/api/settings/dian", bytes.NewBufferString(`{"environment":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	})

	t.Run("Validation_MissingFile", func(t *testing.T) {
		fakeUC := &fakeDIANSettingsUseCase{}
		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		handler := NewSettingsHandler(fakeUC)
		app.Put("/api/settings/dian", func(c *fiber.Ctx) error {
			c.Locals(LocalCompanyID, "company-1")
			return c.Next()
		}, handler.UpdateDIANSettings)

		req, contentType := multipartRequest(t, "/api/settings/dian", "test", "secret123", "", nil)
		req.Header.Set("Content-Type", contentType)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Validation_InvalidExtension", func(t *testing.T) {
		fakeUC := &fakeDIANSettingsUseCase{}
		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		handler := NewSettingsHandler(fakeUC)
		app.Put("/api/settings/dian", func(c *fiber.Ctx) error {
			c.Locals(LocalCompanyID, "company-1")
			return c.Next()
		}, handler.UpdateDIANSettings)

		req, _ := multipartRequest(t, "/api/settings/dian", "test", "secret123", "certificado.pem", []byte("dummy"))
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	})

	t.Run("Validation_TooLarge", func(t *testing.T) {
		fakeUC := &fakeDIANSettingsUseCase{}
		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		handler := NewSettingsHandler(fakeUC)
		app.Put("/api/settings/dian", func(c *fiber.Ctx) error {
			c.Locals(LocalCompanyID, "company-1")
			return c.Next()
		}, handler.UpdateDIANSettings)

		big := bytes.Repeat([]byte("a"), int(maxDIANCertificateSizeBytes)+1)
		req, _ := multipartRequest(t, "/api/settings/dian", "test", "secret123", "certificado.p12", big)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
	})

	t.Run("UseCase_InvalidInput", func(t *testing.T) {
		fakeUC := &fakeDIANSettingsUseCase{saveFunc: func(companyID string, in dto.UpsertDIANSettingsRequest) (*dto.DIANSettingsResponse, error) {
			return nil, domain.ErrInvalidInput
		}}
		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		handler := NewSettingsHandler(fakeUC)
		app.Put("/api/settings/dian", func(c *fiber.Ctx) error {
			c.Locals(LocalCompanyID, "company-1")
			return c.Next()
		}, handler.UpdateDIANSettings)

		req, _ := multipartRequest(t, "/api/settings/dian", "test", "secret123", "certificado.p12", []byte("dummy"))
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("UseCase_Internal", func(t *testing.T) {
		fakeUC := &fakeDIANSettingsUseCase{saveFunc: func(companyID string, in dto.UpsertDIANSettingsRequest) (*dto.DIANSettingsResponse, error) {
			return nil, errors.New("db error")
		}}
		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		handler := NewSettingsHandler(fakeUC)
		app.Put("/api/settings/dian", func(c *fiber.Ctx) error {
			c.Locals(LocalCompanyID, "company-1")
			return c.Next()
		}, handler.UpdateDIANSettings)

		req, _ := multipartRequest(t, "/api/settings/dian", "test", "secret123", "certificado.p12", []byte("dummy"))
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}
