package http

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// RegisterMovementUseCase interfaz local para registrar movimientos de inventario.
type RegisterMovementUseCase interface {
	RegisterMovementFromRequest(ctx context.Context, companyID, userID string, in dto.RegisterMovementRequest) error
	RegisterAdjustmentFromRequest(ctx context.Context, companyID, userID string, in dto.RegisterMovementRequest) (string, error)
}

// ReplenishmentUseCase interfaz local para generar listas de reposición.
type ReplenishmentUseCase interface {
	GenerateReplenishmentList(ctx context.Context, companyID, warehouseID string) ([]dto.ReplenishmentSuggestionDTO, error)
}

// GetStockUseCase interfaz local para obtener resumen de stock.
type GetStockUseCase interface {
	Execute(ctx context.Context, companyID, productID, warehouseID string) (*dto.StockSummaryDTO, error)
}

// ListMovementsUseCase interfaz local para listar movimientos con filtros.
type ListMovementsUseCase interface {
	Execute(ctx context.Context, companyID string, in dto.MovementFiltersDTO) (*dto.PaginatedMovementsDTO, error)
}

// InventoryHandler maneja las peticiones HTTP de movimientos e inventario (protegido).
type InventoryHandler struct {
	uc            RegisterMovementUseCase
	replenishment ReplenishmentUseCase
	getStock      GetStockUseCase
	listMovements ListMovementsUseCase
}

// NewInventoryHandler construye el handler.
func NewInventoryHandler(uc RegisterMovementUseCase, replenishment ReplenishmentUseCase, getStock GetStockUseCase, listMovements ListMovementsUseCase) *InventoryHandler {
	return &InventoryHandler{uc: uc, replenishment: replenishment, getStock: getStock, listMovements: listMovements}
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

// RegisterAdjustment godoc
// @Summary      Registrar ajuste de inventario
// @Description  Registra un movimiento de tipo ADJUSTMENT con razón obligatoria (MERMA|ROBO|VENCIMIENTO|CONTEO_FISICO|DETERIORO|OTRO).
// @Tags         inventory
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body  dto.RegisterMovementRequest  true  "product_id, warehouse_id, quantity (±), adjustment_reason"
// @Success      201   {object}  map[string]string  "{ \"movement_id\": \"uuid\" }"
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Router       /api/inventory/adjustments [post]
func (h *InventoryHandler) RegisterAdjustment(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.RegisterMovementRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	movementID, err := h.uc.RegisterAdjustmentFromRequest(c.Context(), companyID, userID, in)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: err.Error()})
		}
		if errors.Is(err, domain.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "producto o bodega no encontrado"})
		}
		if errors.Is(err, domain.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado al recurso"})
		}
		if errors.Is(err, domain.ErrInsufficientStock) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "INSUFFICIENT_STOCK", Message: "stock insuficiente"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"movement_id": movementID})
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
// @Failure      404  {object}  dto.ErrorResponse
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
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "producto o bodega no encontrado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.JSON(summary)
}

// ListMovements godoc
// @Summary      Listar movimientos de inventario
// @Description  Devuelve movimientos paginados con filtros por producto, bodega, tipo y rango de fechas.
// @Tags         inventory
// @Security     Bearer
// @Produce      json
// @Param        product_id    query  string  false  "ID de producto"
// @Param        warehouse_id  query  string  false  "ID de bodega"
// @Param        type          query  string  false  "Tipo de movimiento (IN|OUT|ADJUSTMENT|TRANSFER|RETURN)"
// @Param        start_date    query  string  false  "Fecha inicio (YYYY-MM-DD)"
// @Param        end_date      query  string  false  "Fecha fin (YYYY-MM-DD)"
// @Param        limit         query  int     false  "Límite" default(20)
// @Param        offset        query  int     false  "Offset" default(0)
// @Success      200  {object}  dto.PaginatedMovementsDTO
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      503  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/inventory/movements [get]
func (h *InventoryHandler) ListMovements(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.listMovements == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "SERVICE_UNAVAILABLE", Message: "listado de movimientos no configurado"})
	}

	parseDate := func(s string) (time.Time, error) {
		if s == "" {
			return time.Time{}, nil
		}
		return time.Parse("2006-01-02", s)
	}

	startDate, err := parseDate(c.Query("start_date"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "start_date inválida (use YYYY-MM-DD)"})
	}
	endDate, err := parseDate(c.Query("end_date"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "end_date inválida (use YYYY-MM-DD)"})
	}

	out, err := h.listMovements.Execute(c.Context(), companyID, dto.MovementFiltersDTO{
		ProductID:   c.Query("product_id"),
		WarehouseID: c.Query("warehouse_id"),
		Type:        c.Query("type"),
		StartDate:   startDate,
		EndDate:     endDate,
		Limit:       c.QueryInt("limit", 20),
		Offset:      c.QueryInt("offset", 0),
	})
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "filtros inválidos"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.JSON(out)
}
