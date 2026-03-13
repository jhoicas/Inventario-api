package http

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	appinventory "github.com/jhoicas/Inventario-api/internal/application/inventory"
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

// StocktakeUseCase interfaz local para conteos físicos (stocktake).
type StocktakeUseCase interface {
	CreateSnapshot(ctx context.Context, companyID, warehouseID string) (string, error)
	UpdateCounts(ctx context.Context, stocktakeID string, items []appinventory.StocktakeItemInput) error
	Close(ctx context.Context, stocktakeID string) error
}

// ReorderConfigUseCase interfaz local para configurar puntos de reposición por producto y bodega.
type ReorderConfigUseCase interface {
	Execute(ctx context.Context, companyID string, in dto.ReorderConfigRequest) error
}

// InventoryHandler maneja las peticiones HTTP de movimientos e inventario (protegido).
type InventoryHandler struct {
	uc            RegisterMovementUseCase
	replenishment ReplenishmentUseCase
	getStock      GetStockUseCase
	listMovements ListMovementsUseCase
	stocktake     StocktakeUseCase
	reorderConfig ReorderConfigUseCase
}

// NewInventoryHandler construye el handler.

func NewInventoryHandler(uc RegisterMovementUseCase, replenishment ReplenishmentUseCase, getStock GetStockUseCase, listMovements ListMovementsUseCase, options ...any) *InventoryHandler {
	h := &InventoryHandler{uc: uc, replenishment: replenishment, getStock: getStock, listMovements: listMovements}
	for _, opt := range options {
		switch v := opt.(type) {
		case StocktakeUseCase:
			h.stocktake = v
		case ReorderConfigUseCase:
			h.reorderConfig = v
		}
	}
	return h
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

type createStocktakeRequest struct {
	WarehouseID string `json:"warehouse_id"`
}

type updateStocktakeCountsRequest struct {
	Items []appinventory.StocktakeItemInput `json:"items"`
}

// CreateStocktakeSnapshot godoc
// @Summary      Crear snapshot de conteo físico
// @Description  Copia el stock actual de la bodega y abre un stocktake en estado OPEN.
// @Tags         inventory
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body  createStocktakeRequest  true  "warehouse_id"
// @Success      201   {object}  map[string]string  "{ \"stocktake_id\": \"uuid\" }"
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      503   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /api/inventory/stocktake [post]
func (h *InventoryHandler) CreateStocktakeSnapshot(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.stocktake == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "SERVICE_UNAVAILABLE", Message: "stocktake no configurado"})
	}

	var in createStocktakeRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}

	stocktakeID, err := h.stocktake.CreateSnapshot(c.Context(), companyID, in.WarehouseID)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if errors.Is(err, domain.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "bodega no encontrada"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"stocktake_id": stocktakeID})
}

// UpdateStocktakeCounts godoc
// @Summary      Actualizar conteo físico
// @Description  Actualiza cantidades contadas del stocktake y recalcula diferencias.
// @Tags         inventory
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id    path  string                       true  "stocktake_id"
// @Param        body  body  updateStocktakeCountsRequest true  "items"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Failure      503   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /api/inventory/stocktake/{id} [put]
func (h *InventoryHandler) UpdateStocktakeCounts(c *fiber.Ctx) error {
	if GetCompanyID(c) == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.stocktake == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "SERVICE_UNAVAILABLE", Message: "stocktake no configurado"})
	}

	stocktakeID := c.Params("id")
	var in updateStocktakeCountsRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}

	if err := h.stocktake.UpdateCounts(c.Context(), stocktakeID, in.Items); err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if errors.Is(err, domain.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "stocktake o item no encontrado"})
		}
		if errors.Is(err, domain.ErrConflict) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "CONFLICT", Message: "stocktake cerrado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.JSON(fiber.Map{"message": "conteo actualizado"})
}

// CloseStocktake godoc
// @Summary      Cerrar conteo físico
// @Description  Cierra el stocktake y genera movimientos ADJUSTMENT por cada diferencia != 0.
// @Tags         inventory
// @Security     Bearer
// @Produce      json
// @Param        id  path  string  true  "stocktake_id"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      409  {object}  dto.ErrorResponse
// @Failure      503  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/inventory/stocktake/{id}/close [post]
func (h *InventoryHandler) CloseStocktake(c *fiber.Ctx) error {
	if GetCompanyID(c) == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.stocktake == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "SERVICE_UNAVAILABLE", Message: "stocktake no configurado"})
	}

	stocktakeID := c.Params("id")
	if err := h.stocktake.Close(c.Context(), stocktakeID); err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if errors.Is(err, domain.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "stocktake no encontrado"})
		}
		if errors.Is(err, domain.ErrConflict) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "CONFLICT", Message: "stocktake ya cerrado"})
		}
		if errors.Is(err, domain.ErrInsufficientStock) {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "INSUFFICIENT_STOCK", Message: "stock insuficiente"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.JSON(fiber.Map{"message": "stocktake cerrado"})
}

// UpdateReorderConfig godoc
// @Summary      Configurar reposición por producto
// @Description  Upsert de configuración en product_reorder_config por producto y bodega.
// @Tags         inventory
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "ID del producto"
// @Param        body  body  dto.ReorderConfigRequest  true  "warehouse_id, reorder_point, min_stock, max_stock, lead_time_days"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      503   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /api/products/{id}/reorder-config [put]
func (h *InventoryHandler) UpdateReorderConfig(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.reorderConfig == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "SERVICE_UNAVAILABLE", Message: "reorder_config no configurado"})
	}

	productID := c.Params("id")
	if productID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id es requerido"})
	}

	var in dto.ReorderConfigRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	in.ProductID = productID

	err := h.reorderConfig.Execute(c.Context(), companyID, in)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		}
		if errors.Is(err, domain.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "producto no encontrado"})
		}
		if errors.Is(err, domain.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado al recurso"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	return c.JSON(fiber.Map{"message": "configuración de reposición actualizada"})
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
