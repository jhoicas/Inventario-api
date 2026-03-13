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

type StocktakeItemInput struct {
	ProductID  string          `json:"product_id"`
	CountedQty decimal.Decimal `json:"counted_qty"`
}

type StocktakeUseCase struct {
	stocktakeRepo StocktakeRepository
	snapshotRepo  StockSnapshotRepository
	txRunner      TxRunner
	registerUC    *RegisterMovementUseCase
}

func NewStocktakeUseCase(
	stocktakeRepo StocktakeRepository,
	snapshotRepo StockSnapshotRepository,
	txRunner TxRunner,
	registerUC *RegisterMovementUseCase,
) *StocktakeUseCase {
	return &StocktakeUseCase{
		stocktakeRepo: stocktakeRepo,
		snapshotRepo:  snapshotRepo,
		txRunner:      txRunner,
		registerUC:    registerUC,
	}
}

// CreateSnapshot crea una sesión de conteo físico copiando el stock actual de una bodega.
func (uc *StocktakeUseCase) CreateSnapshot(ctx context.Context, companyID, warehouseID string) (string, error) {
	if companyID == "" || warehouseID == "" {
		return "", domain.ErrInvalidInput
	}
	if uc.stocktakeRepo == nil || uc.snapshotRepo == nil {
		return "", domain.ErrInvalidInput
	}

	stocks, err := uc.snapshotRepo.ListByWarehouse(ctx, companyID, warehouseID)
	if err != nil {
		return "", err
	}

	now := time.Now()
	stocktakeID := uuid.New().String()
	stocktake := &entity.Stocktake{
		ID:          stocktakeID,
		CompanyID:   companyID,
		WarehouseID: warehouseID,
		Status:      entity.StocktakeStatusOpen,
		CreatedAt:   now,
	}

	items := make([]entity.StocktakeItem, 0, len(stocks))
	for _, s := range stocks {
		if s == nil {
			continue
		}
		items = append(items, entity.StocktakeItem{
			ID:          uuid.New().String(),
			StocktakeID: stocktakeID,
			ProductID:   s.ProductID,
			SystemQty:   s.Quantity,
			CountedQty:  s.Quantity,
			Difference:  decimal.Zero,
		})
	}

	if err := uc.stocktakeRepo.Create(ctx, stocktake, items); err != nil {
		return "", err
	}

	return stocktakeID, nil
}

// UpdateCounts actualiza cantidades contadas y su diferencia contra el snapshot.
func (uc *StocktakeUseCase) UpdateCounts(ctx context.Context, stocktakeID string, items []StocktakeItemInput) error {
	if stocktakeID == "" || len(items) == 0 {
		return domain.ErrInvalidInput
	}
	if uc.stocktakeRepo == nil {
		return domain.ErrInvalidInput
	}

	st, err := uc.stocktakeRepo.GetByID(ctx, stocktakeID)
	if err != nil {
		return err
	}
	if st == nil {
		return domain.ErrNotFound
	}
	if st.Status != entity.StocktakeStatusOpen {
		return domain.ErrConflict
	}

	existing, err := uc.stocktakeRepo.ListItems(ctx, stocktakeID)
	if err != nil {
		return err
	}
	index := make(map[string]entity.StocktakeItem, len(existing))
	for _, it := range existing {
		index[it.ProductID] = it
	}

	toUpdate := make([]entity.StocktakeItem, 0, len(items))
	for _, in := range items {
		if in.ProductID == "" {
			return domain.ErrInvalidInput
		}
		base, ok := index[in.ProductID]
		if !ok {
			return domain.ErrNotFound
		}
		base.CountedQty = in.CountedQty
		base.Difference = in.CountedQty.Sub(base.SystemQty)
		toUpdate = append(toUpdate, base)
	}

	return uc.stocktakeRepo.UpdateCounts(ctx, stocktakeID, toUpdate)
}

// Close cierra el conteo y genera movimientos ADJUSTMENT por cada diferencia != 0.
// Reutiliza RegisterMovementUseCase en una transacción compartida mediante TxRunner.
func (uc *StocktakeUseCase) Close(ctx context.Context, stocktakeID string) error {
	if stocktakeID == "" {
		return domain.ErrInvalidInput
	}
	if uc.stocktakeRepo == nil || uc.txRunner == nil || uc.registerUC == nil {
		return domain.ErrInvalidInput
	}

	st, err := uc.stocktakeRepo.GetByID(ctx, stocktakeID)
	if err != nil {
		return err
	}
	if st == nil {
		return domain.ErrNotFound
	}
	if st.Status != entity.StocktakeStatusOpen {
		return domain.ErrConflict
	}

	items, err := uc.stocktakeRepo.ListItems(ctx, stocktakeID)
	if err != nil {
		return err
	}

	now := time.Now()
	txID := uuid.New().String()
	err = uc.txRunner.Run(ctx, func(
		movRepo repository.InventoryMovementRepository,
		stockRepo repository.StockRepository,
		productRepo repository.ProductRepository,
	) error {
		for _, it := range items {
			if it.Difference.IsZero() {
				continue
			}

			product, pErr := productRepo.GetByID(it.ProductID)
			if pErr != nil || product == nil {
				return domain.ErrNotFound
			}
			if product.CompanyID != st.CompanyID {
				return domain.ErrForbidden
			}

			input := MovementInputDTO{
				CompanyID:        st.CompanyID,
				UserID:           "",
				ProductID:        it.ProductID,
				WarehouseID:      st.WarehouseID,
				Type:             string(entity.MovementTypeADJUSTMENT),
				Quantity:         it.Difference,
				UnitCost:         &product.Cost,
				AdjustmentReason: "CONTEO_FISICO",
				Notes:            "CONTEO_FISICO",
			}

			if err := uc.registerUC.doADJUSTMENT(movRepo, stockRepo, productRepo, product, input, now, txID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return uc.stocktakeRepo.MarkClosed(ctx, stocktakeID, now)
}
