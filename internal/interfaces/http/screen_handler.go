package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// ScreenUseCase define el contrato para CRUD de pantallas del catálogo.
type ScreenUseCase interface {
	GetScreens(ctx context.Context) ([]dto.ScreenAdminResponse, error)
	CreateScreen(ctx context.Context, in dto.CreateScreenRequest) (*dto.ScreenAdminResponse, error)
	UpdateScreen(ctx context.Context, id string, in dto.UpdateScreenRequest) (*dto.ScreenAdminResponse, error)
}

// ScreenHandler expone endpoints de super admin para screens.
type ScreenHandler struct {
	uc ScreenUseCase
}

// NewScreenHandler construye el handler.
func NewScreenHandler(uc ScreenUseCase) *ScreenHandler {
	return &ScreenHandler{uc: uc}
}

// List lista todas las pantallas del catálogo.
func (h *ScreenHandler) List(c *fiber.Ctx) error {
	out, err := h.uc.GetScreens(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// Create crea una pantalla.
func (h *ScreenHandler) Create(c *fiber.Ctx) error {
	var in dto.CreateScreenRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.uc.CreateScreen(c.Context(), in)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		case domain.ErrDuplicate:
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "la pantalla ya existe"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// Update actualiza una pantalla por ID.
func (h *ScreenHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "id es requerido"})
	}
	var in dto.UpdateScreenRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.uc.UpdateScreen(c.Context(), id, in)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "pantalla no encontrada"})
		case domain.ErrDuplicate:
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "la pantalla ya existe"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}
