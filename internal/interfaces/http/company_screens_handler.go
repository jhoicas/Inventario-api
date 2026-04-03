package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// CompanyScreenUseCase define el contrato mínimo para administrar pantallas por empresa.
type CompanyScreenUseCase interface {
	ListScreens(ctx context.Context, companyID string) (*dto.CompanyScreensResponse, error)
	UpsertScreen(ctx context.Context, companyID string, in dto.CreateCompanyScreenRequest) (*dto.CompanyScreenResponse, error)
	UpdateScreen(ctx context.Context, companyID, screenID string, in dto.UpdateCompanyScreenRequest) (*dto.CompanyScreenResponse, error)
	DeleteScreen(ctx context.Context, companyID, screenID string) error
}

// CompanyScreenHandler expone endpoints para habilitar/deshabilitar pantallas por empresa.
type CompanyScreenHandler struct {
	uc CompanyScreenUseCase
}

// NewCompanyScreenHandler construye el handler.
func NewCompanyScreenHandler(uc CompanyScreenUseCase) *CompanyScreenHandler {
	return &CompanyScreenHandler{uc: uc}
}

// List devuelve las pantallas configuradas para una empresa.
func (h *CompanyScreenHandler) List(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	out, err := h.uc.ListScreens(c.Context(), companyID)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "company_id inválido"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "empresa no encontrada"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// Upsert crea o actualiza la configuración de una pantalla para la empresa.
func (h *CompanyScreenHandler) Upsert(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	var in dto.CreateCompanyScreenRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.uc.UpsertScreen(c.Context(), companyID, in)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "empresa o pantalla no encontrada"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// Update actualiza el estado de una pantalla para la empresa.
func (h *CompanyScreenHandler) Update(c *fiber.Ctx) error {
	companyID := c.Params("id")
	screenID := c.Params("screen_id")
	if companyID == "" || screenID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_PARAMS", Message: "id y screen_id son requeridos"})
	}
	var in dto.UpdateCompanyScreenRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.uc.UpdateScreen(c.Context(), companyID, screenID, in)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "empresa o pantalla no encontrada"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// Delete desactiva una pantalla para la empresa.
func (h *CompanyScreenHandler) Delete(c *fiber.Ctx) error {
	companyID := c.Params("id")
	screenID := c.Params("screen_id")
	if companyID == "" || screenID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_PARAMS", Message: "id y screen_id son requeridos"})
	}
	if err := h.uc.DeleteScreen(c.Context(), companyID, screenID); err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "empresa o pantalla no encontrada"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
