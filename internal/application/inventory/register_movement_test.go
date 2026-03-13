package inventory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// ── Constantes y datos base ────────────────────────────────────────────────────

const (
	testCompanyID   = "company-123"
	testUserID      = "user-456"
	testProductID   = "product-789"
	testWarehouseID = "warehouse-abc"
)

// ── Fake TxRunner ──────────────────────────────────────────────────────────────

type fakeTxRunner struct {
	runFunc func(ctx context.Context, fn func(
		movRepo repository.InventoryMovementRepository,
		stockRepo repository.StockRepository,
		productRepo repository.ProductRepository,
	) error) error
}

func (f *fakeTxRunner) Run(ctx context.Context, fn func(
	movRepo repository.InventoryMovementRepository,
	stockRepo repository.StockRepository,
	productRepo repository.ProductRepository,
) error) error {
	if f.runFunc != nil {
		return f.runFunc(ctx, fn)
	}
	return nil
}

var _ TxRunner = (*fakeTxRunner)(nil)

// ── Fake ProductRepository ────────────────────────────────────────────────────

type fakeProductRepo struct {
	getByIDFunc            func(id string) (*entity.Product, error)
	updateCostFunc         func(productID string, cost decimal.Decimal) error
	createFunc             func(product *entity.Product) error
	getByCompanyAndSKUFunc func(companyID, sku string) (*entity.Product, error)
	updateFunc             func(product *entity.Product) error
	listByCompanyFunc      func(companyID string, limit, offset int) ([]*entity.Product, error)
	deleteFunc             func(id string) error
}

func (f *fakeProductRepo) Create(product *entity.Product) error {
	if f.createFunc != nil {
		return f.createFunc(product)
	}
	return nil
}
func (f *fakeProductRepo) GetByID(id string) (*entity.Product, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(id)
	}
	return nil, nil
}
func (f *fakeProductRepo) GetByCompanyAndSKU(companyID, sku string) (*entity.Product, error) {
	if f.getByCompanyAndSKUFunc != nil {
		return f.getByCompanyAndSKUFunc(companyID, sku)
	}
	return nil, nil
}
func (f *fakeProductRepo) Update(product *entity.Product) error {
	if f.updateFunc != nil {
		return f.updateFunc(product)
	}
	return nil
}
func (f *fakeProductRepo) UpdateCost(productID string, cost decimal.Decimal) error {
	if f.updateCostFunc != nil {
		return f.updateCostFunc(productID, cost)
	}
	return nil
}
func (f *fakeProductRepo) ListByCompany(companyID string, limit, offset int) ([]*entity.Product, error) {
	if f.listByCompanyFunc != nil {
		return f.listByCompanyFunc(companyID, limit, offset)
	}
	return nil, nil
}
func (f *fakeProductRepo) Delete(id string) error {
	if f.deleteFunc != nil {
		return f.deleteFunc(id)
	}
	return nil
}

var _ repository.ProductRepository = (*fakeProductRepo)(nil)

// ── Fake WarehouseRepository ──────────────────────────────────────────────────

type fakeWarehouseRepo struct {
	getByIDFunc       func(id string) (*entity.Warehouse, error)
	createFunc        func(warehouse *entity.Warehouse) error
	updateFunc        func(warehouse *entity.Warehouse) error
	listByCompanyFunc func(companyID string, limit, offset int) ([]*entity.Warehouse, error)
	deleteFunc        func(id string) error
}

func (f *fakeWarehouseRepo) Create(warehouse *entity.Warehouse) error {
	if f.createFunc != nil {
		return f.createFunc(warehouse)
	}
	return nil
}
func (f *fakeWarehouseRepo) GetByID(id string) (*entity.Warehouse, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(id)
	}
	return nil, nil
}
func (f *fakeWarehouseRepo) Update(warehouse *entity.Warehouse) error {
	if f.updateFunc != nil {
		return f.updateFunc(warehouse)
	}
	return nil
}
func (f *fakeWarehouseRepo) ListByCompany(companyID string, limit, offset int) ([]*entity.Warehouse, error) {
	if f.listByCompanyFunc != nil {
		return f.listByCompanyFunc(companyID, limit, offset)
	}
	return nil, nil
}
func (f *fakeWarehouseRepo) Delete(id string) error {
	if f.deleteFunc != nil {
		return f.deleteFunc(id)
	}
	return nil
}

var _ repository.WarehouseRepository = (*fakeWarehouseRepo)(nil)

// ── Fake InventoryMovementRepository ───────────────────────────────────────────

type fakeMovementRepo struct {
	createFunc          func(movement *entity.InventoryMovement) error
	getByIDFunc         func(id string) (*entity.InventoryMovement, error)
	listFunc            func(companyID string, f repository.MovementFilters) ([]*entity.InventoryMovement, int64, error)
	listByWarehouseFunc func(warehouseID string, from, to *time.Time, limit, offset int) ([]*entity.InventoryMovement, error)
	listByProductFunc   func(productID string, from, to *time.Time, limit, offset int) ([]*entity.InventoryMovement, error)
}

func (f *fakeMovementRepo) Create(movement *entity.InventoryMovement) error {
	if f.createFunc != nil {
		return f.createFunc(movement)
	}
	return nil
}
func (f *fakeMovementRepo) GetByID(id string) (*entity.InventoryMovement, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(id)
	}
	return nil, nil
}
func (f *fakeMovementRepo) List(companyID string, flt repository.MovementFilters) ([]*entity.InventoryMovement, int64, error) {
	if f.listFunc != nil {
		return f.listFunc(companyID, flt)
	}
	return nil, 0, nil
}
func (f *fakeMovementRepo) ListByWarehouse(warehouseID string, from, to *time.Time, limit, offset int) ([]*entity.InventoryMovement, error) {
	if f.listByWarehouseFunc != nil {
		return f.listByWarehouseFunc(warehouseID, from, to, limit, offset)
	}
	return nil, nil
}
func (f *fakeMovementRepo) ListByProduct(productID string, from, to *time.Time, limit, offset int) ([]*entity.InventoryMovement, error) {
	if f.listByProductFunc != nil {
		return f.listByProductFunc(productID, from, to, limit, offset)
	}
	return nil, nil
}

var _ repository.InventoryMovementRepository = (*fakeMovementRepo)(nil)

// ── Fake StockRepository ───────────────────────────────────────────────────────

type fakeStockRepo struct {
	getFunc          func(productID, warehouseID string) (*entity.Stock, error)
	getByProductFunc func(productID string) ([]*entity.Stock, error)
	getSummaryFunc   func(productID, warehouseID string) (*repository.StockSummary, error)
	getForUpdateFunc func(productID, warehouseID string) (*entity.Stock, error)
	upsertFunc       func(stock *entity.Stock) error
}

func (f *fakeStockRepo) Get(productID, warehouseID string) (*entity.Stock, error) {
	if f.getFunc != nil {
		return f.getFunc(productID, warehouseID)
	}
	return nil, nil
}
func (f *fakeStockRepo) GetByProduct(productID string) ([]*entity.Stock, error) {
	if f.getByProductFunc != nil {
		return f.getByProductFunc(productID)
	}
	return nil, nil
}
func (f *fakeStockRepo) GetSummary(productID, warehouseID string) (*repository.StockSummary, error) {
	if f.getSummaryFunc != nil {
		return f.getSummaryFunc(productID, warehouseID)
	}
	return nil, nil
}
func (f *fakeStockRepo) GetForUpdate(productID, warehouseID string) (*entity.Stock, error) {
	if f.getForUpdateFunc != nil {
		return f.getForUpdateFunc(productID, warehouseID)
	}
	return nil, nil
}
func (f *fakeStockRepo) Upsert(stock *entity.Stock) error {
	if f.upsertFunc != nil {
		return f.upsertFunc(stock)
	}
	return nil
}

var _ repository.StockRepository = (*fakeStockRepo)(nil)

// ── Helpers ────────────────────────────────────────────────────────────────────

func validProduct(companyID string) *entity.Product {
	return &entity.Product{
		ID:        testProductID,
		CompanyID: companyID,
		SKU:       "SKU-001",
		Name:      "Producto Test",
		Cost:      decimal.NewFromInt(5000),
	}
}

func validWarehouse(companyID string) *entity.Warehouse {
	return &entity.Warehouse{
		ID:        testWarehouseID,
		CompanyID: companyID,
		Name:      "Bodega Central",
	}
}

func validStock(qty decimal.Decimal) *entity.Stock {
	return &entity.Stock{
		ProductID:   testProductID,
		WarehouseID: testWarehouseID,
		Quantity:    qty,
		UpdatedAt:   time.Now(),
	}
}

// validRegisterMovementDTO devuelve un MovementInputDTO base para entrada (IN).
func validRegisterMovementDTO() MovementInputDTO {
	unitCost := decimal.NewFromInt(5000)
	return MovementInputDTO{
		CompanyID:   testCompanyID,
		UserID:      testUserID,
		ProductID:   testProductID,
		WarehouseID: testWarehouseID,
		Type:        string(entity.MovementTypeIN),
		Quantity:    decimal.NewFromInt(10),
		UnitCost:    &unitCost,
	}
}

// validRegisterMovementRequest devuelve el DTO HTTP base (para RegisterMovementFromRequest).
func validRegisterMovementRequest() dto.RegisterMovementRequest {
	unitCost := decimal.NewFromInt(5000)
	return dto.RegisterMovementRequest{
		ProductID:   testProductID,
		WarehouseID: testWarehouseID,
		Type:        string(entity.MovementTypeIN),
		Quantity:    decimal.NewFromInt(10),
		UnitCost:    &unitCost,
	}
}

// ── Tests RegisterMovement (Table-Driven) ───────────────────────────────────────

func TestRegisterMovementUseCase_RegisterMovement(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		input      MovementInputDTO
		setup      func() (TxRunner, *fakeProductRepo, *fakeWarehouseRepo)
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:  "Success_IN",
			input: validRegisterMovementDTO(),
			setup: func() (TxRunner, *fakeProductRepo, *fakeWarehouseRepo) {
				productRepo := &fakeProductRepo{
					getByIDFunc: func(id string) (*entity.Product, error) {
						if id != testProductID {
							return nil, nil
						}
						return validProduct(testCompanyID), nil
					},
					updateCostFunc: func(productID string, cost decimal.Decimal) error {
						assert.Equal(t, testProductID, productID)
						return nil
					},
				}
				warehouseRepo := &fakeWarehouseRepo{
					getByIDFunc: func(id string) (*entity.Warehouse, error) {
						if id != testWarehouseID {
							return nil, nil
						}
						return validWarehouse(testCompanyID), nil
					},
				}
				movRepo := &fakeMovementRepo{
					createFunc: func(m *entity.InventoryMovement) error {
						assert.Equal(t, entity.MovementTypeIN, m.Type)
						assert.True(t, m.Quantity.Equal(decimal.NewFromInt(10)))
						return nil
					},
				}
				stockRepo := &fakeStockRepo{
					getForUpdateFunc: func(productID, warehouseID string) (*entity.Stock, error) {
						return validStock(decimal.Zero), nil
					},
					upsertFunc: func(stock *entity.Stock) error {
						assert.True(t, stock.Quantity.Equal(decimal.NewFromInt(10)))
						return nil
					},
				}
				txRunner := &fakeTxRunner{
					runFunc: func(_ context.Context, fn func(
						repository.InventoryMovementRepository,
						repository.StockRepository,
						repository.ProductRepository,
					) error) error {
						return fn(movRepo, stockRepo, productRepo)
					},
				}
				return txRunner, productRepo, warehouseRepo
			},
			wantErr: nil,
		},
		{
			name: "Success_OUT",
			input: func() MovementInputDTO {
				d := validRegisterMovementDTO()
				d.Type = string(entity.MovementTypeOUT)
				d.UnitCost = nil
				d.Quantity = decimal.NewFromInt(3)
				return d
			}(),
			setup: func() (TxRunner, *fakeProductRepo, *fakeWarehouseRepo) {
				productRepo := &fakeProductRepo{
					getByIDFunc: func(id string) (*entity.Product, error) {
						return validProduct(testCompanyID), nil
					},
				}
				warehouseRepo := &fakeWarehouseRepo{
					getByIDFunc: func(id string) (*entity.Warehouse, error) {
						return validWarehouse(testCompanyID), nil
					},
				}
				movRepo := &fakeMovementRepo{
					createFunc: func(m *entity.InventoryMovement) error {
						assert.Equal(t, entity.MovementTypeOUT, m.Type)
						assert.True(t, m.Quantity.Equal(decimal.NewFromInt(-3)))
						return nil
					},
				}
				stockRepo := &fakeStockRepo{
					getForUpdateFunc: func(_, _ string) (*entity.Stock, error) {
						return validStock(decimal.NewFromInt(10)), nil
					},
					upsertFunc: func(stock *entity.Stock) error {
						assert.True(t, stock.Quantity.Equal(decimal.NewFromInt(7)))
						return nil
					},
				}
				txRunner := &fakeTxRunner{
					runFunc: func(_ context.Context, fn func(
						repository.InventoryMovementRepository,
						repository.StockRepository,
						repository.ProductRepository,
					) error) error {
						return fn(movRepo, stockRepo, productRepo)
					},
				}
				return txRunner, productRepo, warehouseRepo
			},
			wantErr: nil,
		},
		{
			name: "InsufficientStock_OUT",
			input: func() MovementInputDTO {
				d := validRegisterMovementDTO()
				d.Type = string(entity.MovementTypeOUT)
				d.UnitCost = nil
				d.Quantity = decimal.NewFromInt(100)
				return d
			}(),
			setup: func() (TxRunner, *fakeProductRepo, *fakeWarehouseRepo) {
				productRepo := &fakeProductRepo{
					getByIDFunc: func(id string) (*entity.Product, error) {
						return validProduct(testCompanyID), nil
					},
				}
				warehouseRepo := &fakeWarehouseRepo{
					getByIDFunc: func(id string) (*entity.Warehouse, error) {
						return validWarehouse(testCompanyID), nil
					},
				}
				movRepo := &fakeMovementRepo{}
				stockRepo := &fakeStockRepo{
					getForUpdateFunc: func(_, _ string) (*entity.Stock, error) {
						return validStock(decimal.NewFromInt(5)), nil
					},
				}
				txRunner := &fakeTxRunner{
					runFunc: func(_ context.Context, fn func(
						repository.InventoryMovementRepository,
						repository.StockRepository,
						repository.ProductRepository,
					) error) error {
						return fn(movRepo, stockRepo, productRepo)
					},
				}
				return txRunner, productRepo, warehouseRepo
			},
			wantErr: domain.ErrInsufficientStock,
		},
		{
			name:  "NotFound_Product",
			input: validRegisterMovementDTO(),
			setup: func() (TxRunner, *fakeProductRepo, *fakeWarehouseRepo) {
				productRepo := &fakeProductRepo{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						return nil, nil
					},
				}
				warehouseRepo := &fakeWarehouseRepo{
					getByIDFunc: func(id string) (*entity.Warehouse, error) {
						return validWarehouse(testCompanyID), nil
					},
				}
				txRunner := &fakeTxRunner{runFunc: func(_ context.Context, fn func(
					repository.InventoryMovementRepository,
					repository.StockRepository,
					repository.ProductRepository,
				) error) error {
					return fn(nil, nil, productRepo)
				}}
				return txRunner, productRepo, warehouseRepo
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name:  "NotFound_Warehouse",
			input: validRegisterMovementDTO(),
			setup: func() (TxRunner, *fakeProductRepo, *fakeWarehouseRepo) {
				productRepo := &fakeProductRepo{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						return validProduct(testCompanyID), nil
					},
				}
				warehouseRepo := &fakeWarehouseRepo{
					getByIDFunc: func(_ string) (*entity.Warehouse, error) {
						return nil, nil
					},
				}
				txRunner := &fakeTxRunner{}
				return txRunner, productRepo, warehouseRepo
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name:  "Forbidden_ProductFromOtherCompany",
			input: validRegisterMovementDTO(),
			setup: func() (TxRunner, *fakeProductRepo, *fakeWarehouseRepo) {
				productRepo := &fakeProductRepo{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						p := validProduct("otra-empresa")
						return p, nil
					},
				}
				warehouseRepo := &fakeWarehouseRepo{
					getByIDFunc: func(_ string) (*entity.Warehouse, error) {
						return validWarehouse(testCompanyID), nil
					},
				}
				return &fakeTxRunner{}, productRepo, warehouseRepo
			},
			wantErr: domain.ErrForbidden,
		},
		{
			name: "InvalidInput_EmptyProductID",
			input: func() MovementInputDTO {
				d := validRegisterMovementDTO()
				d.ProductID = ""
				return d
			}(),
			setup: func() (TxRunner, *fakeProductRepo, *fakeWarehouseRepo) {
				return &fakeTxRunner{}, &fakeProductRepo{}, &fakeWarehouseRepo{}
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "InvalidInput_IN_NoUnitCost",
			input: func() MovementInputDTO {
				d := validRegisterMovementDTO()
				d.UnitCost = nil
				return d
			}(),
			setup: func() (TxRunner, *fakeProductRepo, *fakeWarehouseRepo) {
				productRepo := &fakeProductRepo{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						return validProduct(testCompanyID), nil
					},
				}
				warehouseRepo := &fakeWarehouseRepo{
					getByIDFunc: func(_ string) (*entity.Warehouse, error) {
						return validWarehouse(testCompanyID), nil
					},
				}
				return &fakeTxRunner{}, productRepo, warehouseRepo
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:  "RepoError_StockGetForUpdateFails",
			input: validRegisterMovementDTO(),
			setup: func() (TxRunner, *fakeProductRepo, *fakeWarehouseRepo) {
				productRepo := &fakeProductRepo{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						return validProduct(testCompanyID), nil
					},
				}
				warehouseRepo := &fakeWarehouseRepo{
					getByIDFunc: func(_ string) (*entity.Warehouse, error) {
						return validWarehouse(testCompanyID), nil
					},
				}
				stockRepo := &fakeStockRepo{
					getForUpdateFunc: func(_, _ string) (*entity.Stock, error) {
						return nil, errors.New("db: row lock timeout")
					},
				}
				txRunner := &fakeTxRunner{
					runFunc: func(_ context.Context, fn func(
						repository.InventoryMovementRepository,
						repository.StockRepository,
						repository.ProductRepository,
					) error) error {
						return fn(&fakeMovementRepo{}, stockRepo, productRepo)
					},
				}
				return txRunner, productRepo, warehouseRepo
			},
			wantAnyErr: true,
		},
		{
			name:  "RepoError_MovementCreateFails",
			input: validRegisterMovementDTO(),
			setup: func() (TxRunner, *fakeProductRepo, *fakeWarehouseRepo) {
				productRepo := &fakeProductRepo{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						return validProduct(testCompanyID), nil
					},
					updateCostFunc: func(_ string, _ decimal.Decimal) error { return nil },
				}
				warehouseRepo := &fakeWarehouseRepo{
					getByIDFunc: func(_ string) (*entity.Warehouse, error) {
						return validWarehouse(testCompanyID), nil
					},
				}
				movRepo := &fakeMovementRepo{
					createFunc: func(_ *entity.InventoryMovement) error {
						return errors.New("db: constraint violation")
					},
				}
				stockRepo := &fakeStockRepo{
					getForUpdateFunc: func(_, _ string) (*entity.Stock, error) {
						return validStock(decimal.Zero), nil
					},
					upsertFunc: func(_ *entity.Stock) error { return nil },
				}
				txRunner := &fakeTxRunner{
					runFunc: func(_ context.Context, fn func(
						repository.InventoryMovementRepository,
						repository.StockRepository,
						repository.ProductRepository,
					) error) error {
						return fn(movRepo, stockRepo, productRepo)
					},
				}
				return txRunner, productRepo, warehouseRepo
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txRunner, productRepo, warehouseRepo := tt.setup()
			uc := NewRegisterMovementUseCase(txRunner, productRepo, warehouseRepo)

			err := uc.RegisterMovement(ctx, tt.input)

			if tt.wantAnyErr {
				require.Error(t, err)
				return
			}
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// ── Tests RegisterMovementFromRequest ──────────────────────────────────────────

func TestRegisterMovementUseCase_RegisterMovementFromRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("Success_AdaptsDTO", func(t *testing.T) {
		productRepo := &fakeProductRepo{
			getByIDFunc: func(_ string) (*entity.Product, error) {
				return validProduct(testCompanyID), nil
			},
			updateCostFunc: func(_ string, _ decimal.Decimal) error { return nil },
		}
		warehouseRepo := &fakeWarehouseRepo{
			getByIDFunc: func(_ string) (*entity.Warehouse, error) {
				return validWarehouse(testCompanyID), nil
			},
		}
		movRepo := &fakeMovementRepo{createFunc: func(_ *entity.InventoryMovement) error { return nil }}
		stockRepo := &fakeStockRepo{
			getForUpdateFunc: func(_, _ string) (*entity.Stock, error) { return validStock(decimal.Zero), nil },
			upsertFunc:       func(_ *entity.Stock) error { return nil },
		}
		txRunner := &fakeTxRunner{
			runFunc: func(_ context.Context, fn func(
				repository.InventoryMovementRepository,
				repository.StockRepository,
				repository.ProductRepository,
			) error) error {
				return fn(movRepo, stockRepo, productRepo)
			},
		}
		uc := NewRegisterMovementUseCase(txRunner, productRepo, warehouseRepo)

		err := uc.RegisterMovementFromRequest(ctx, testCompanyID, testUserID, validRegisterMovementRequest())
		require.NoError(t, err)
	})
}
