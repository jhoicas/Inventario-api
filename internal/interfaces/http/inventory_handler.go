package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// RegisterMovementUseCase interfaz local para registrar movimientos de inventario.
type RegisterMovementUseCase interface {
	RegisterMovementFromRequest(ctx context.Context, companyID, userID string, in dto.RegisterMovementRequest) error
}

// ReplenishmentUseCase interfaz local para generar listas de reposición.
type ReplenishmentUseCase interface {
	GenerateReplenishmentList(ctx context.Context, companyID, warehouseID string) ([]dto.ReplenishmentSuggestionDTO, error)
}

// GetStockUseCase interfaz local para obtener resumen de stock.
type GetStockUseCase interface {
	Execute(ctx context.Context, companyID, productID, warehouseID string) (*dto.StockSummaryDTO, error)
}

// InventoryHandler maneja las peticiones HTTP de movimientos e inventario (protegido).
type InventoryHandler struct {
	uc            RegisterMovementUseCase
	replenishment ReplenishmentUseCase
	getStock      GetStockUseCase
}

// NewInventoryHandler construye el handler.
func NewInventoryHandler(uc RegisterMovementUseCase, replenishment ReplenishmentUseCase, getStock GetStockUseCase) *InventoryHandler {
	return &InventoryHandler{uc: uc, replenishment: replenishment, getStock: getStock}
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

// GetReplenishmentList godoc
// @Summary      Lista semanal de reposición
// @Description  Devuelve los SKUs por debajo del punto de reorden con la cantidad sugerida
//
//	de pedido, ordenados por margen histórico y volumen de ventas.
//
// @Tags         inventory
// @Security     Bearer
// @Produce      json
// @Param        warehouse_id  query  string  false  "Filtrar por bodega (UUID). Vacío = stock global."
// @Success      200  {array}   dto.ReplenishmentSuggestionDTO
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/inventory/replenishment-list [get]
func (h *InventoryHandler) GetReplenishmentList(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}

	warehouseID := c.Query("warehouse_id")

	list, err := h.replenishment.GenerateReplenishmentList(c.Context(), companyID, warehouseID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.JSON(fiber.Map{
		"total":          len(list),
		"replenishments": list,
	})
}

// GetStock godoc
// @Summary      Resumen de stock
// @Description  Devuelve el resumen de stock de un producto en una bodega o agregado de todas las bodegas.
// @Tags         inventory
// @Security     Bearer
// @Produce      json
// @Param        product_id   query  string  true   "ID del producto (UUID)"
// @Param        warehouse_id query  string  false  "ID de la bodega (UUID). Vacío = stock agregado de todas las bodegas."
// @Success      200  {object}  dto.StockSummaryDTO
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/inventory/stock [get]
func (h *InventoryHandler) GetStock(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}

	productID := c.Query("product_id")
	if productID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "product_id es requerido"})
	}

	warehouseID := c.Query("warehouse_id")

	summary, err := h.getStock.Execute(c.Context(), companyID, productID, warehouseID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.JSON(summary)
}
