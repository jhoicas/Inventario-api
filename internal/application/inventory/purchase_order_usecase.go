package inventory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/shopspring/decimal"
)

type PurchaseOrderItemInput struct {
	ProductID string          `json:"product_id"`
	Quantity  decimal.Decimal `json:"quantity"`
	UnitCost  decimal.Decimal `json:"unit_cost"`
}

type CreatePurchaseOrderInput struct {
	SupplierID string                   `json:"supplier_id"`
	Number     string                   `json:"number"`
	Date       time.Time                `json:"date"`
	Items      []PurchaseOrderItemInput `json:"items"`
}

// PurchaseOrderUseCase gestiona órdenes de compra y su recepción en inventario.
type PurchaseOrderUseCase struct {
	poRepo             PurchaseOrderRepository
	supplierRepo       repository.SupplierRepository
	warehouseRepo      repository.WarehouseRepository
	txRunner           TxRunner
	registerMovementUC *RegisterMovementUseCase
}

func NewPurchaseOrderUseCase(
	poRepo PurchaseOrderRepository,
	supplierRepo repository.SupplierRepository,
	warehouseRepo repository.WarehouseRepository,
	txRunner TxRunner,
	registerMovementUC *RegisterMovementUseCase,
) *PurchaseOrderUseCase {
	return &PurchaseOrderUseCase{
		poRepo:             poRepo,
		supplierRepo:       supplierRepo,
		warehouseRepo:      warehouseRepo,
		txRunner:           txRunner,
		registerMovementUC: registerMovementUC,
	}
}

func (uc *PurchaseOrderUseCase) Create(ctx context.Context, companyID string, in CreatePurchaseOrderInput) (string, error) {
	if companyID == "" || in.SupplierID == "" || len(in.Items) == 0 {
		return "", domain.ErrInvalidInput
	}

	supplier, err := uc.supplierRepo.GetByID(in.SupplierID)
	if err != nil {
		return "", err
	}
	if supplier == nil {
		return "", domain.ErrNotFound
	}
	if supplier.CompanyID != companyID {
		return "", domain.ErrForbidden
	}

	items := make([]entity.PurchaseOrderItem, 0, len(in.Items))
	for _, item := range in.Items {
		if item.ProductID == "" || !item.Quantity.GreaterThan(decimal.Zero) || item.UnitCost.LessThan(decimal.Zero) {
			return "", domain.ErrInvalidInput
		}
		items = append(items, entity.PurchaseOrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitCost:  item.UnitCost,
		})
	}

	now := time.Now()
	poNumber := in.Number
	if poNumber == "" {
		poNumber = "PO-" + now.Format("20060102150405")
	}
	poDate := in.Date
	if poDate.IsZero() {
		poDate = now
	}

	po := &entity.PurchaseOrder{
		ID:         uuid.New().String(),
		CompanyID:  companyID,
		SupplierID: in.SupplierID,
		Number:     poNumber,
		Date:       poDate,
		Status:     entity.PurchaseOrderStatusDraft,
		Items:      items,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := uc.poRepo.Create(ctx, po); err != nil {
		return "", err
	}

	return po.ID, nil
}

func (uc *PurchaseOrderUseCase) ListByCompany(ctx context.Context, companyID string, limit, offset int) ([]*entity.PurchaseOrder, int64, error) {
	if companyID == "" {
		return nil, 0, domain.ErrInvalidInput
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return uc.poRepo.ListByCompany(ctx, companyID, limit, offset)
}

func (uc *PurchaseOrderUseCase) UpdateStatus(ctx context.Context, companyID, purchaseOrderID, status string) error {
	if companyID == "" || purchaseOrderID == "" {
		return domain.ErrInvalidInput
	}
	if !isValidPurchaseOrderStatus(status) {
		return domain.ErrInvalidInput
	}

	po, err := uc.poRepo.GetByID(ctx, purchaseOrderID)
	if err != nil {
		return err
	}
	if po == nil {
		return domain.ErrNotFound
	}
	if po.CompanyID != companyID {
		return domain.ErrForbidden
	}

	return uc.poRepo.UpdateStatus(ctx, purchaseOrderID, status, time.Now())
}

// Receive registra movimientos IN por cada item de la orden de compra en una sola transacción.
// Si falla cualquier IN, toda la recepción hace rollback.
func (uc *PurchaseOrderUseCase) Receive(ctx context.Context, companyID, userID, purchaseOrderID, warehouseID string) error {
	if companyID == "" || userID == "" || purchaseOrderID == "" || warehouseID == "" {
		return domain.ErrInvalidInput
	}

	po, err := uc.poRepo.GetByID(ctx, purchaseOrderID)
	if err != nil {
		return err
	}
	if po == nil {
		return domain.ErrNotFound
	}
	if po.CompanyID != companyID {
		return domain.ErrForbidden
	}
	if po.Status == entity.PurchaseOrderStatusClosed {
		return domain.ErrConflict
	}
	if len(po.Items) == 0 {
		return domain.ErrInvalidInput
	}

	wh, err := uc.warehouseRepo.GetByID(warehouseID)
	if err != nil {
		return err
	}
	if wh == nil {
		return domain.ErrNotFound
	}
	if wh.CompanyID != companyID {
		return domain.ErrForbidden
	}

	now := time.Now()
	txID := uuid.New().String()
	err = uc.txRunner.Run(ctx, func(
		movRepo repository.InventoryMovementRepository,
		stockRepo repository.StockRepository,
		productRepo repository.ProductRepository,
	) error {
		for _, item := range po.Items {
			if item.ProductID == "" || !item.Quantity.GreaterThan(decimal.Zero) || item.UnitCost.LessThan(decimal.Zero) {
				return domain.ErrInvalidInput
			}

			product, pErr := productRepo.GetByID(item.ProductID)
			if pErr != nil {
				return pErr
			}
			if product == nil {
				return domain.ErrNotFound
			}
			if product.CompanyID != companyID {
				return domain.ErrForbidden
			}

			unitCost := item.UnitCost
			input := MovementInputDTO{
				CompanyID:   companyID,
				UserID:      userID,
				ProductID:   item.ProductID,
				WarehouseID: warehouseID,
				Type:        string(entity.MovementTypeIN),
				Quantity:    item.Quantity,
				UnitCost:    &unitCost,
				Notes:       "PO:" + po.ID,
			}

			if err := uc.registerMovementUC.doIN(movRepo, stockRepo, productRepo, product, input, now, txID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return uc.poRepo.UpdateStatus(ctx, po.ID, entity.PurchaseOrderStatusClosed, time.Now())
}

func isValidPurchaseOrderStatus(status string) bool {
	switch status {
	case entity.PurchaseOrderStatusDraft,
		entity.PurchaseOrderStatusSent,
		entity.PurchaseOrderStatusConfirmed,
		entity.PurchaseOrderStatusPartialReceipt,
		entity.PurchaseOrderStatusClosed:
		return true
	default:
		return false
	}
}
