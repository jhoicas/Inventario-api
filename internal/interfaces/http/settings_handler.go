package http

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

const maxDIANCertificateSizeBytes int64 = 5 * 1024 * 1024

type DIANSettingsUseCase interface {
	Save(companyID string, in dto.UpsertDIANSettingsRequest) (*dto.DIANSettingsResponse, error)
}

type SettingsHandler struct {
	uc DIANSettingsUseCase
}

func NewSettingsHandler(uc DIANSettingsUseCase) *SettingsHandler {
	return &SettingsHandler{uc: uc}
}

// UpdateDIANSettings godoc
// @Summary      Guardar configuración DIAN con certificado .p12
// @Tags         settings
// @Security     Bearer
// @Accept       mpfd
// @Produce      json
// @Param        environment           formData  string true  "Entorno DIAN: test|prod"
// @Param        certificate_password  formData  string true  "Contraseña del certificado .p12"
// @Param        certificate           formData  file   true  "Archivo .p12"
// @Success      200  {object}  dto.DIANSettingsResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      413  {object}  dto.ErrorResponse
// @Failure      415  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/settings/dian [put]
func (h *SettingsHandler) UpdateDIANSettings(c *fiber.Ctx) error {
	if h.uc == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "SERVICE_UNAVAILABLE", Message: "servicio no disponible"})
	}

	contentType := strings.ToLower(c.Get("Content-Type"))
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		return c.Status(fiber.StatusUnsupportedMediaType).JSON(dto.ErrorResponse{Code: "UNSUPPORTED_MEDIA_TYPE", Message: "Content-Type debe ser multipart/form-data"})
	}

	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}

	environment := strings.TrimSpace(strings.ToLower(c.FormValue("environment")))
	if environment == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "environment es requerido"})
	}

	certificatePassword := c.FormValue("certificate_password")
	if strings.TrimSpace(certificatePassword) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "certificate_password es requerido"})
	}

	fileHeader, err := c.FormFile("certificate")
	if err != nil || fileHeader == nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "certificate es requerido"})
	}
	if fileHeader.Size <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "certificate vacío"})
	}
	if fileHeader.Size > maxDIANCertificateSizeBytes {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(dto.ErrorResponse{Code: "PAYLOAD_TOO_LARGE", Message: "certificate supera el tamaño máximo permitido"})
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".p12" {
		return c.Status(fiber.StatusUnsupportedMediaType).JSON(dto.ErrorResponse{Code: "UNSUPPORTED_MEDIA_TYPE", Message: "solo se permite archivo .p12"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_FILE", Message: "no se pudo abrir certificate"})
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_FILE", Message: "no se pudo leer certificate"})
	}
	if int64(len(fileData)) > maxDIANCertificateSizeBytes {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(dto.ErrorResponse{Code: "PAYLOAD_TOO_LARGE", Message: "certificate supera el tamaño máximo permitido"})
	}

	out, err := h.uc.Save(companyID, dto.UpsertDIANSettingsRequest{
		Environment:         environment,
		CertificateFileName: fileHeader.Filename,
		CertificateData:     fileData,
		CertificatePassword: certificatePassword,
	})
	if err != nil {
		switch err {
		case domain.ErrUnauthorized:
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
		case domain.ErrForbidden:
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "empresa no encontrada"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: "no se pudo guardar configuración DIAN"})
		}
	}

	return c.JSON(out)
}
