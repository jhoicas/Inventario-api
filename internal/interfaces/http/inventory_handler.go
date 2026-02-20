package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/application/inventory"
	"github.com/tu-usuario/inventory-pro/internal/domain"
)

// InventoryHandler maneja las peticiones HTTP de movimientos de inventario (protegido).
type InventoryHandler struct {
	uc *inventory.RegisterMovementUseCase
}

// NewInventoryHandler construye el handler.
func NewInventoryHandler(uc *inventory.RegisterMovementUseCase) *InventoryHandler {
	return &InventoryHandler{uc: uc}
}

// RegisterMovement godoc
// @Summary      Registrar movimiento de inventario
// @Tags         inventory
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body  dto.RegisterMovementRequest  true  "product_id, warehouse_id (o from/to para TRANSFER), type, quantity, unit_cost (entradas)"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Router       /api/inventory/movements [post]
func (h *InventoryHandler) RegisterMovement(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.RegisterMovementRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	err := h.uc.RegisterMovementFromRequest(c.Context(), companyID, userID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "producto o bodega no encontrado"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado al recurso"})
		}
		if err == domain.ErrInsufficientStock {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "INSUFFICIENT_STOCK", Message: "stock insuficiente"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "movimiento registrado"})
}
