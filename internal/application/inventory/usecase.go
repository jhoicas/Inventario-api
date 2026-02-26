package inventory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/inventory"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// RegisterMovementUseCase registra movimientos de inventario de forma transaccional
// (IN, OUT, ADJUSTMENT, TRANSFER) con bloqueo de fila (SELECT FOR UPDATE) y Commit/Rollback.
type RegisterMovementUseCase struct {
	txRunner      TxRunner
	productRepo   repository.ProductRepository
	warehouseRepo repository.WarehouseRepository
}

// NewRegisterMovementUseCase construye el caso de uso.
func NewRegisterMovementUseCase(
	txRunner TxRunner,
	productRepo repository.ProductRepository,
	warehouseRepo repository.WarehouseRepository,
) *RegisterMovementUseCase {
	return &RegisterMovementUseCase{
		txRunner:      txRunner,
		productRepo:   productRepo,
		warehouseRepo: warehouseRepo,
	}
}

// MovementInputDTO entrada para Registrar un movimiento de inventario.
// Para IN/OUT/ADJUSTMENT: ProductID, WarehouseID, Type, Quantity; UnitCost obligatorio en IN.
// Para TRANSFER: ProductID, FromWarehouseID, ToWarehouseID, Type=TRANSFER, Quantity.
type MovementInputDTO struct {
	CompanyID       string
	UserID          string
	ProductID       string
	WarehouseID     string
	FromWarehouseID string
	ToWarehouseID   string
	Type            string
	Quantity        decimal.Decimal
	UnitCost        *decimal.Decimal
}

// RegisterMovement inicia una transacción, bloquea la fila en inventory_stock (SELECT FOR UPDATE),
// aplica la lógica según tipo (IN/OUT/TRANSFER/ADJUSTMENT) y hace Commit o Rollback.
func (uc *RegisterMovementUseCase) RegisterMovement(ctx context.Context, input MovementInputDTO) error {
	// Validar tipo y campos
	switch input.Type {
	case entity.MovementTypeIN, entity.MovementTypeOUT, entity.MovementTypeADJUSTMENT:
		if input.ProductID == "" || input.WarehouseID == "" {
			return domain.ErrInvalidInput
		}
		if input.Quantity.IsZero() {
			return domain.ErrInvalidInput
		}
		if input.Type == entity.MovementTypeIN && (input.UnitCost == nil || input.UnitCost.LessThan(decimal.Zero)) {
			return domain.ErrInvalidInput
		}
		if input.Type == entity.MovementTypeOUT && input.Quantity.LessThan(decimal.Zero) {
			return domain.ErrInvalidInput
		}
	case entity.MovementTypeTRANSFER:
		if input.ProductID == "" || input.FromWarehouseID == "" || input.ToWarehouseID == "" {
			return domain.ErrInvalidInput
		}
		if input.FromWarehouseID == input.ToWarehouseID || !input.Quantity.GreaterThan(decimal.Zero) {
			return domain.ErrInvalidInput
		}
	default:
		return domain.ErrInvalidInput
	}

	// Validar que producto y bodega(s) existan y sean de la empresa
	product, err := uc.productRepo.GetByID(input.ProductID)
	if err != nil || product == nil {
		return domain.ErrNotFound
	}
	if product.CompanyID != input.CompanyID {
		return domain.ErrForbidden
	}

	if input.Type == entity.MovementTypeTRANSFER {
		fromWh, _ := uc.warehouseRepo.GetByID(input.FromWarehouseID)
		toWh, _ := uc.warehouseRepo.GetByID(input.ToWarehouseID)
		if fromWh == nil || toWh == nil || fromWh.CompanyID != input.CompanyID || toWh.CompanyID != input.CompanyID {
			return domain.ErrNotFound
		}
	} else {
		wh, _ := uc.warehouseRepo.GetByID(input.WarehouseID)
		if wh == nil || wh.CompanyID != input.CompanyID {
			return domain.ErrNotFound
		}
	}

	now := time.Now()
	txID := uuid.New().String()

	// Inicia transacción; Commit si todo ok, Rollback si algo falla (TxRunner.Run lo hace)
	return uc.txRunner.Run(ctx, func(
		movRepo repository.InventoryMovementRepository,
		stockRepo repository.StockRepository,
		productRepo repository.ProductRepository,
	) error {
		switch input.Type {
		case entity.MovementTypeIN:
			return uc.doIN(movRepo, stockRepo, productRepo, product, input, now, txID)
		case entity.MovementTypeOUT:
			return uc.doOUT(movRepo, stockRepo, productRepo, product, input, now, txID)
		case entity.MovementTypeADJUSTMENT:
			return uc.doADJUSTMENT(movRepo, stockRepo, productRepo, product, input, now, txID)
		case entity.MovementTypeTRANSFER:
			return uc.doTRANSFER(movRepo, stockRepo, productRepo, input, now, txID)
		}
		return domain.ErrInvalidInput
	})
}

// doIN: bloquea fila (GetForUpdate), CostCalculator, actualiza costo producto, suma stock, guarda movimiento.
func (uc *RegisterMovementUseCase) doIN(
	movRepo repository.InventoryMovementRepository,
	stockRepo repository.StockRepository,
	productRepo repository.ProductRepository,
	product *entity.Product,
	input MovementInputDTO,
	now time.Time, txID string,
) error {
	// Bloquea la fila en inventory_stock (SELECT FOR UPDATE) para evitar condiciones de carrera
	stock, err := stockRepo.GetForUpdate(input.ProductID, input.WarehouseID)
	if err != nil {
		return err
	}
	unitCost := *input.UnitCost
	newQty := stock.Quantity.Add(input.Quantity)
	newCost := inventory.CostCalculator(stock.Quantity, product.Cost, input.Quantity, unitCost)

	// Actualiza costo del producto en products
	if err := productRepo.UpdateCost(input.ProductID, newCost); err != nil {
		return err
	}
	// Suma la cantidad al stock
	stock.Quantity = newQty
	stock.UpdatedAt = now
	if err := stockRepo.Upsert(stock); err != nil {
		return err
	}
	// Guarda registro en inventory_movements
	mov := &entity.InventoryMovement{
		TransactionID: txID,
		ProductID:     input.ProductID,
		WarehouseID:   input.WarehouseID,
		Type:          entity.MovementTypeIN,
		Quantity:      input.Quantity,
		UnitCost:      unitCost,
		TotalCost:     input.Quantity.Mul(unitCost),
		Date:          now,
		CreatedAt:     now,
		CreatedBy:     input.UserID,
	}
	return movRepo.Create(mov)
}

// RegisterOUTInTx ejecuta una salida (OUT) usando los repositorios proporcionados (misma transacción del caller).
// Implementa la interfaz billing.InventoryUseCase para integración facturación-inventario.
// ctx propaga la transacción SQL; transactionID suele ser el ID de la factura.
func (uc *RegisterMovementUseCase) RegisterOUTInTx(
	ctx context.Context,
	movRepo repository.InventoryMovementRepository,
	stockRepo repository.StockRepository,
	_ repository.ProductRepository,
	product *entity.Product,
	productID, warehouseID, userID string,
	quantity decimal.Decimal,
	now time.Time,
	transactionID string,
) error {
	stock, err := stockRepo.GetForUpdate(productID, warehouseID)
	if err != nil {
		return err
	}
	if stock.Quantity.LessThan(quantity) {
		return domain.ErrInsufficientStock
	}
	stock.Quantity = stock.Quantity.Sub(quantity)
	stock.UpdatedAt = now
	if err := stockRepo.Upsert(stock); err != nil {
		return err
	}
	unitCost := product.Cost
	mov := &entity.InventoryMovement{
		TransactionID: transactionID,
		ProductID:     productID,
		WarehouseID:   warehouseID,
		Type:          entity.MovementTypeOUT,
		Quantity:      quantity.Neg(),
		UnitCost:      unitCost,
		TotalCost:     quantity.Neg().Mul(unitCost),
		Date:          now,
		CreatedAt:     now,
		CreatedBy:     userID,
	}
	return movRepo.Create(mov)
}

// doOUT: bloquea fila, verifica StockActual >= CantidadSolicitada, resta cantidad, guarda movimiento al costo promedio actual.
func (uc *RegisterMovementUseCase) doOUT(
	movRepo repository.InventoryMovementRepository,
	stockRepo repository.StockRepository,
	_ repository.ProductRepository,
	product *entity.Product,
	input MovementInputDTO,
	now time.Time, txID string,
) error {
	// Bloquea la fila en inventory_stock (SELECT FOR UPDATE)
	stock, err := stockRepo.GetForUpdate(input.ProductID, input.WarehouseID)
	if err != nil {
		return err
	}
	if stock.Quantity.LessThan(input.Quantity) {
		return domain.ErrInsufficientStock
	}
	stock.Quantity = stock.Quantity.Sub(input.Quantity)
	stock.UpdatedAt = now
	if err := stockRepo.Upsert(stock); err != nil {
		return err
	}
	unitCost := product.Cost
	mov := &entity.InventoryMovement{
		TransactionID: txID,
		ProductID:     input.ProductID,
		WarehouseID:   input.WarehouseID,
		Type:          entity.MovementTypeOUT,
		Quantity:      input.Quantity.Neg(),
		UnitCost:      unitCost,
		TotalCost:     input.Quantity.Neg().Mul(unitCost),
		Date:          now,
		CreatedAt:     now,
		CreatedBy:     input.UserID,
	}
	return movRepo.Create(mov)
}

// doADJUSTMENT: positivo como IN, negativo como OUT.
func (uc *RegisterMovementUseCase) doADJUSTMENT(
	movRepo repository.InventoryMovementRepository,
	stockRepo repository.StockRepository,
	productRepo repository.ProductRepository,
	product *entity.Product,
	input MovementInputDTO,
	now time.Time, txID string,
) error {
	if input.Quantity.GreaterThan(decimal.Zero) {
		unitCost := decimal.Zero
		if input.UnitCost != nil {
			unitCost = *input.UnitCost
		}
		input.UnitCost = &unitCost
		return uc.doIN(movRepo, stockRepo, productRepo, product, input, now, txID)
	}
	adjOut := input
	adjOut.Quantity = input.Quantity.Neg()
	return uc.doOUT(movRepo, stockRepo, productRepo, product, adjOut, now, txID)
}

// doTRANSFER: resta de bodega origen, suma en bodega destino, misma transacción; guarda dos registros en inventory_movements.
func (uc *RegisterMovementUseCase) doTRANSFER(
	movRepo repository.InventoryMovementRepository,
	stockRepo repository.StockRepository,
	productRepo repository.ProductRepository,
	input MovementInputDTO,
	now time.Time, txID string,
) error {
	// Bloquea fila en bodega origen
	origin, err := stockRepo.GetForUpdate(input.ProductID, input.FromWarehouseID)
	if err != nil {
		return err
	}
	if origin.Quantity.LessThan(input.Quantity) {
		return domain.ErrInsufficientStock
	}
	dest, _ := stockRepo.Get(input.ProductID, input.ToWarehouseID)
	if dest == nil {
		dest = &entity.Stock{ProductID: input.ProductID, WarehouseID: input.ToWarehouseID, Quantity: decimal.Zero, UpdatedAt: now}
	}
	// Resta de bodega origen y suma en bodega destino (misma transacción)
	origin.Quantity = origin.Quantity.Sub(input.Quantity)
	dest.Quantity = dest.Quantity.Add(input.Quantity)
	origin.UpdatedAt = now
	dest.UpdatedAt = now
	if err := stockRepo.Upsert(origin); err != nil {
		return err
	}
	if err := stockRepo.Upsert(dest); err != nil {
		return err
	}
	product, err := productRepo.GetByID(input.ProductID)
	if err != nil || product == nil {
		return domain.ErrNotFound
	}
	unitCost := product.Cost
	// Guarda movimiento salida en origen
	outMov := &entity.InventoryMovement{
		TransactionID: txID,
		ProductID:     input.ProductID,
		WarehouseID:   input.FromWarehouseID,
		Type:          entity.MovementTypeTRANSFER,
		Quantity:      input.Quantity.Neg(),
		UnitCost:      unitCost,
		TotalCost:     input.Quantity.Neg().Mul(unitCost),
		Date:          now,
		CreatedAt:     now,
		CreatedBy:     input.UserID,
	}
	if err := movRepo.Create(outMov); err != nil {
		return err
	}
	// Guarda movimiento entrada en destino
	inMov := &entity.InventoryMovement{
		TransactionID: txID,
		ProductID:     input.ProductID,
		WarehouseID:   input.ToWarehouseID,
		Type:          entity.MovementTypeTRANSFER,
		Quantity:      input.Quantity,
		UnitCost:      unitCost,
		TotalCost:     input.Quantity.Mul(unitCost),
		Date:          now,
		CreatedAt:     now,
		CreatedBy:     input.UserID,
	}
	return movRepo.Create(inMov)
}
