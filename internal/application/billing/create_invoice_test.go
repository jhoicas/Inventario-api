package billing

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

// ── Constantes ────────────────────────────────────────────────────────────────

const (
	testCompanyID   = "company-123"
	testUserID      = "user-456"
	testCustomerID  = "customer-789"
	testWarehouseID = "warehouse-abc"
	testProductID1  = "product-001"
	testProductID2  = "product-002"
)

// ── Fake BillingTxRunner ───────────────────────────────────────────────────────

type fakeBillingTxRunner struct {
	runFunc func(ctx context.Context, fn func(
		movRepo repository.InventoryMovementRepository,
		stockRepo repository.StockRepository,
		productRepo repository.ProductRepository,
		customerRepo repository.CustomerRepository,
		invoiceRepo repository.InvoiceRepository,
	) error) error
}

func (f *fakeBillingTxRunner) RunBilling(ctx context.Context, fn func(
	movRepo repository.InventoryMovementRepository,
	stockRepo repository.StockRepository,
	productRepo repository.ProductRepository,
	customerRepo repository.CustomerRepository,
	invoiceRepo repository.InvoiceRepository,
) error) error {
	if f.runFunc != nil {
		return f.runFunc(ctx, fn)
	}
	return nil
}

var _ BillingTxRunner = (*fakeBillingTxRunner)(nil)

// ── Fake InventoryUseCase ──────────────────────────────────────────────────────

type fakeInventoryUC struct {
	registerOUTFunc    func(ctx context.Context, movRepo repository.InventoryMovementRepository, stockRepo repository.StockRepository, productRepo repository.ProductRepository, product *entity.Product, productID, warehouseID, userID string, quantity decimal.Decimal, now time.Time, transactionID string) error
	registerReturnFunc func(ctx context.Context, movRepo repository.InventoryMovementRepository, stockRepo repository.StockRepository, productRepo repository.ProductRepository, product *entity.Product, productID, warehouseID, userID string, quantity decimal.Decimal, now time.Time, transactionID string) error
}

func (f *fakeInventoryUC) RegisterOUTInTx(
	ctx context.Context,
	movRepo repository.InventoryMovementRepository,
	stockRepo repository.StockRepository,
	productRepo repository.ProductRepository,
	product *entity.Product,
	productID, warehouseID, userID string,
	quantity decimal.Decimal,
	now time.Time,
	transactionID string,
) error {
	if f.registerOUTFunc != nil {
		return f.registerOUTFunc(ctx, movRepo, stockRepo, productRepo, product, productID, warehouseID, userID, quantity, now, transactionID)
	}
	return nil
}

func (f *fakeInventoryUC) RegisterReturnInTx(
	ctx context.Context,
	movRepo repository.InventoryMovementRepository,
	stockRepo repository.StockRepository,
	productRepo repository.ProductRepository,
	product *entity.Product,
	productID, warehouseID, userID string,
	quantity decimal.Decimal,
	now time.Time,
	transactionID string,
) error {
	if f.registerReturnFunc != nil {
		return f.registerReturnFunc(ctx, movRepo, stockRepo, productRepo, product, productID, warehouseID, userID, quantity, now, transactionID)
	}
	return nil
}

var _ InventoryUseCase = (*fakeInventoryUC)(nil)

// ── Fake CustomerRepository ────────────────────────────────────────────────────

type fakeCustomerRepo struct {
	getByIDFunc             func(id string) (*entity.Customer, error)
	createFunc              func(customer *entity.Customer) error
	getByCompanyAndTaxIDFunc func(companyID, taxID string) (*entity.Customer, error)
	listByCompanyFunc       func(companyID string, limit, offset int) ([]*entity.Customer, error)
	updateFunc              func(customer *entity.Customer) error
	deleteFunc              func(id string) error
}

func (f *fakeCustomerRepo) Create(customer *entity.Customer) error {
	if f.createFunc != nil {
		return f.createFunc(customer)
	}
	return nil
}
func (f *fakeCustomerRepo) GetByID(id string) (*entity.Customer, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(id)
	}
	return nil, nil
}
func (f *fakeCustomerRepo) GetByCompanyAndTaxID(companyID, taxID string) (*entity.Customer, error) {
	if f.getByCompanyAndTaxIDFunc != nil {
		return f.getByCompanyAndTaxIDFunc(companyID, taxID)
	}
	return nil, nil
}
func (f *fakeCustomerRepo) ListByCompany(companyID string, search string, limit, offset int) ([]*entity.Customer, error) {
	if f.listByCompanyFunc != nil {
		// tests existentes no usan search; se ignora en el fake
		return f.listByCompanyFunc(companyID, limit, offset)
	}
	return nil, nil
}
func (f *fakeCustomerRepo) Update(customer *entity.Customer) error {
	if f.updateFunc != nil {
		return f.updateFunc(customer)
	}
	return nil
}
func (f *fakeCustomerRepo) Delete(id string) error {
	if f.deleteFunc != nil {
		return f.deleteFunc(id)
	}
	return nil
}

var _ repository.CustomerRepository = (*fakeCustomerRepo)(nil)

// ── Fake CompanyRepository ───────────────────────────────────────────────────

type fakeCompanyRepo struct {
	getByIDFunc         func(id string) (*entity.Company, error)
	hasActiveModuleFunc func(ctx context.Context, companyID, moduleName string) (bool, error)
	createFunc          func(company *entity.Company) error
	getByNITFunc        func(nit string) (*entity.Company, error)
	updateFunc         func(company *entity.Company) error
	listFunc            func(limit, offset int) ([]*entity.Company, error)
	deleteFunc         func(id string) error
}

func (f *fakeCompanyRepo) Create(company *entity.Company) error {
	if f.createFunc != nil {
		return f.createFunc(company)
	}
	return nil
}
func (f *fakeCompanyRepo) GetByID(id string) (*entity.Company, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(id)
	}
	return nil, nil
}
func (f *fakeCompanyRepo) GetByNIT(nit string) (*entity.Company, error) {
	if f.getByNITFunc != nil {
		return f.getByNITFunc(nit)
	}
	return nil, nil
}
func (f *fakeCompanyRepo) HasActiveModule(ctx context.Context, companyID, moduleName string) (bool, error) {
	if f.hasActiveModuleFunc != nil {
		return f.hasActiveModuleFunc(ctx, companyID, moduleName)
	}
	return false, nil
}
func (f *fakeCompanyRepo) Update(company *entity.Company) error {
	if f.updateFunc != nil {
		return f.updateFunc(company)
	}
	return nil
}
func (f *fakeCompanyRepo) List(limit, offset int) ([]*entity.Company, error) {
	if f.listFunc != nil {
		return f.listFunc(limit, offset)
	}
	return nil, nil
}
func (f *fakeCompanyRepo) Delete(id string) error {
	if f.deleteFunc != nil {
		return f.deleteFunc(id)
	}
	return nil
}

var _ repository.CompanyRepository = (*fakeCompanyRepo)(nil)

// ── Fake ProductRepository ─────────────────────────────────────────────────────

type fakeProductRepo struct {
	getByIDFunc            func(id string) (*entity.Product, error)
	createFunc             func(product *entity.Product) error
	getByCompanyAndSKUFunc func(companyID, sku string) (*entity.Product, error)
	updateFunc             func(product *entity.Product) error
	updateCostFunc         func(productID string, cost decimal.Decimal) error
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
	getByIDFunc         func(id string) (*entity.Warehouse, error)
	createFunc          func(warehouse *entity.Warehouse) error
	updateFunc          func(warehouse *entity.Warehouse) error
	listByCompanyFunc   func(companyID string, limit, offset int) ([]*entity.Warehouse, error)
	deleteFunc          func(id string) error
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

// ── Fake InvoiceRepository ────────────────────────────────────────────────────

type fakeInvoiceRepo struct {
	createFunc           func(invoice *entity.Invoice) error
	createDetailFunc     func(detail *entity.InvoiceDetail) error
	updateFunc           func(invoice *entity.Invoice) error
	getByIDFunc          func(id string) (*entity.Invoice, error)
	getDetailsByInvoiceIDFunc func(invoiceID string) ([]*entity.InvoiceDetail, error)
	getDIANStatusFunc    func(id string) (*entity.Invoice, error)
	updateReturnStatusFunc func(invoiceID string, status string) error
}

func (f *fakeInvoiceRepo) Create(invoice *entity.Invoice) error {
	if f.createFunc != nil {
		return f.createFunc(invoice)
	}
	return nil
}
func (f *fakeInvoiceRepo) CreateDetail(detail *entity.InvoiceDetail) error {
	if f.createDetailFunc != nil {
		return f.createDetailFunc(detail)
	}
	return nil
}
func (f *fakeInvoiceRepo) Update(invoice *entity.Invoice) error {
	if f.updateFunc != nil {
		return f.updateFunc(invoice)
	}
	return nil
}
func (f *fakeInvoiceRepo) GetByID(id string) (*entity.Invoice, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(id)
	}
	return nil, nil
}
func (f *fakeInvoiceRepo) GetDetailsByInvoiceID(invoiceID string) ([]*entity.InvoiceDetail, error) {
	if f.getDetailsByInvoiceIDFunc != nil {
		return f.getDetailsByInvoiceIDFunc(invoiceID)
	}
	return nil, nil
}
func (f *fakeInvoiceRepo) GetDIANStatus(id string) (*entity.Invoice, error) {
	if f.getDIANStatusFunc != nil {
		return f.getDIANStatusFunc(id)
	}
	return nil, nil
}
func (f *fakeInvoiceRepo) UpdateReturnStatus(invoiceID string, status string) error {
	if f.updateReturnStatusFunc != nil {
		return f.updateReturnStatusFunc(invoiceID, status)
	}
	return nil
}

var _ repository.InvoiceRepository = (*fakeInvoiceRepo)(nil)

// ── Helpers de dominio ────────────────────────────────────────────────────────

func validCustomer(companyID string) *entity.Customer {
	return &entity.Customer{
		ID:        testCustomerID,
		CompanyID: companyID,
		Name:      "Cliente Prueba",
		TaxID:     "900123456",
	}
}

func validCompany(companyID string) *entity.Company {
	return &entity.Company{
		ID:     companyID,
		Name:   "Empresa Test",
		NIT:    "900111222",
		Status: "active",
	}
}

// validProduct devuelve un producto con precio y tasa de IVA (TaxRate 19 = 19%).
func validProduct(companyID, productID string, price, taxRate decimal.Decimal) *entity.Product {
	return &entity.Product{
		ID:        productID,
		CompanyID: companyID,
		SKU:       "SKU-" + productID,
		Name:      "Producto " + productID,
		Price:     price,
		Cost:      decimal.Zero,
		TaxRate:   taxRate,
	}
}

func validWarehouse(companyID string) *entity.Warehouse {
	return &entity.Warehouse{
		ID:        testWarehouseID,
		CompanyID: companyID,
		Name:      "Bodega Central",
	}
}

// validCreateInvoiceRequest devuelve un request con un cliente y dos ítems con cantidades y precios definidos.
// Productos: 001 precio 10000 IVA 19%, 002 precio 50000 IVA 5%.
// Ítem1: 2 x 10000 = 20000 subtotal, tax 19% = 3800.
// Ítem2: 1 x 50000 = 50000 subtotal, tax 5% = 2500.
// NetTotal = 70000, TaxTotal = 6300, GrandTotal = 76300.
func validCreateInvoiceRequest() dto.CreateInvoiceRequest {
	return dto.CreateInvoiceRequest{
		CustomerID:  testCustomerID,
		Prefix:      "FV",
		Number:      "",
		Items: []dto.InvoiceItemRequest{
			{ProductID: testProductID1, Quantity: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(10000)},
			{ProductID: testProductID2, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50000)},
		},
	}
}

// expectTotalsFromValidRequest devuelve los totales esperados para validCreateInvoiceRequest()
// con productos 19% y 5% según validProduct.
func expectTotalsFromValidRequest() (netTotal, taxTotal, grandTotal decimal.Decimal) {
	// toRate: si rate > 1 entonces rate/100, sino rate.
	// Product1 TaxRate 19 -> 0.19; Product2 TaxRate 5 -> 0.05.
	sub1 := decimal.NewFromInt(2).Mul(decimal.NewFromInt(10000)) // 20000
	sub2 := decimal.NewFromInt(1).Mul(decimal.NewFromInt(50000)) // 50000
	netTotal = sub1.Add(sub2)                                    // 70000
	taxTotal = sub1.Mul(decimal.NewFromFloat(0.19)).Add(sub2.Mul(decimal.NewFromFloat(0.05)))
	// taxTotal = 3800 + 2500 = 6300
	grandTotal = netTotal.Add(taxTotal) // 76300
	return netTotal, taxTotal, grandTotal
}

// ── Tests CreateInvoice (Table-Driven) ────────────────────────────────────────

func TestCreateInvoiceUseCase_CreateInvoice(t *testing.T) {
	ctx := context.Background()
	// Sin DIAN en tests: no llamamos ProcessAsync.
	dianConfig := DIANConfig{TechnicalKey: ""}

	tests := []struct {
		name        string
		companyID   string
		userID      string
		in          dto.CreateInvoiceRequest
		setup       func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo)
		wantErr     error
		wantAnyErr  bool
		validateOut func(t *testing.T, out *dto.InvoiceResponse)
	}{
		{
			name:      "Success_NoInventory_TotalsCorrect",
			companyID: testCompanyID,
			userID:    testUserID,
			in:        validCreateInvoiceRequest(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{
					getByIDFunc: func(id string) (*entity.Customer, error) {
						if id != testCustomerID {
							return nil, nil
						}
						return validCustomer(testCompanyID), nil
					},
				}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc: func(id string) (*entity.Company, error) {
						return validCompany(id), nil
					},
					hasActiveModuleFunc: func(_ context.Context, _, moduleName string) (bool, error) {
						assert.Equal(t, entity.ModuleInventory, moduleName)
						return false, nil
					},
				}
				productRepo := &fakeProductRepo{
					getByIDFunc: func(id string) (*entity.Product, error) {
						switch id {
						case testProductID1:
							return validProduct(testCompanyID, testProductID1, decimal.NewFromInt(10000), decimal.NewFromInt(19)), nil
						case testProductID2:
							return validProduct(testCompanyID, testProductID2, decimal.NewFromInt(50000), decimal.NewFromInt(5)), nil
						default:
							return nil, nil
						}
					},
				}
				warehouseRepo := &fakeWarehouseRepo{}
				invoiceRepo := &fakeInvoiceRepo{
					createFunc:       func(inv *entity.Invoice) error { return nil },
					createDetailFunc: func(_ *entity.InvoiceDetail) error { return nil },
				}
				txRunner := &fakeBillingTxRunner{
					runFunc: func(_ context.Context, fn func(
						repository.InventoryMovementRepository,
						repository.StockRepository,
						repository.ProductRepository,
						repository.CustomerRepository,
						repository.InvoiceRepository,
					) error) error {
						return fn(nil, nil, productRepo, customerRepo, invoiceRepo)
					},
				}
				return txRunner, &fakeInventoryUC{}, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo
			},
			wantErr: nil,
			validateOut: func(t *testing.T, out *dto.InvoiceResponse) {
				require.NotNil(t, out)
				require.Len(t, out.Details, 2)
				expectedNet, expectedTax, expectedGrand := expectTotalsFromValidRequest()
				assert.True(t, out.NetTotal.Equal(expectedNet), "NetTotal: got %s want %s", out.NetTotal.String(), expectedNet.String())
				assert.True(t, out.TaxTotal.Equal(expectedTax), "TaxTotal: got %s want %s", out.TaxTotal.String(), expectedTax.String())
				assert.True(t, out.GrandTotal.Equal(expectedGrand), "GrandTotal: got %s want %s", out.GrandTotal.String(), expectedGrand.String())
				// Subtotales por línea
				sub1 := decimal.NewFromInt(2).Mul(decimal.NewFromInt(10000))
				sub2 := decimal.NewFromInt(1).Mul(decimal.NewFromInt(50000))
				assert.True(t, out.Details[0].Subtotal.Equal(sub1), "Detail[0].Subtotal")
				assert.True(t, out.Details[1].Subtotal.Equal(sub2), "Detail[1].Subtotal")
				assert.True(t, out.Details[0].TaxRate.Equal(decimal.NewFromFloat(0.19)), "Detail[0].TaxRate 19%%")
				assert.True(t, out.Details[1].TaxRate.Equal(decimal.NewFromFloat(0.05)), "Detail[1].TaxRate 5%%")
				assert.Equal(t, "Cliente Prueba", out.CustomerName)
			},
		},
		{
			name:      "CustomerNotFound",
			companyID: testCompanyID,
			userID:    testUserID,
			in:        validCreateInvoiceRequest(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{
					getByIDFunc: func(_ string) (*entity.Customer, error) {
						return nil, nil
					},
				}
				companyRepo := &fakeCompanyRepo{getByIDFunc: func(id string) (*entity.Company, error) { return validCompany(id), nil }}
				productRepo := &fakeProductRepo{}
				warehouseRepo := &fakeWarehouseRepo{}
				invoiceRepo := &fakeInvoiceRepo{}
				txRunner := &fakeBillingTxRunner{}
				return txRunner, &fakeInventoryUC{}, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name:      "Forbidden_CustomerFromOtherCompany",
			companyID: testCompanyID,
			userID:    testUserID,
			in:        validCreateInvoiceRequest(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{
					getByIDFunc: func(_ string) (*entity.Customer, error) {
						return validCustomer("otra-empresa"), nil
					},
				}
				companyRepo := &fakeCompanyRepo{getByIDFunc: func(id string) (*entity.Company, error) { return validCompany(id), nil }}
				productRepo := &fakeProductRepo{}
				warehouseRepo := &fakeWarehouseRepo{}
				invoiceRepo := &fakeInvoiceRepo{}
				txRunner := &fakeBillingTxRunner{}
				return txRunner, &fakeInventoryUC{}, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo
			},
			wantErr: domain.ErrForbidden,
		},
		{
			name:      "CompanyNotFound",
			companyID: testCompanyID,
			userID:    testUserID,
			in:        validCreateInvoiceRequest(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{
					getByIDFunc: func(_ string) (*entity.Customer, error) {
						return validCustomer(testCompanyID), nil
					},
				}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc: func(_ string) (*entity.Company, error) {
						return nil, errors.New("not found")
					},
				}
				productRepo := &fakeProductRepo{}
				warehouseRepo := &fakeWarehouseRepo{}
				invoiceRepo := &fakeInvoiceRepo{}
				txRunner := &fakeBillingTxRunner{}
				return txRunner, &fakeInventoryUC{}, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name:      "ProductNotFound",
			companyID: testCompanyID,
			userID:    testUserID,
			in:        validCreateInvoiceRequest(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{
					getByIDFunc: func(_ string) (*entity.Customer, error) {
						return validCustomer(testCompanyID), nil
					},
				}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc: func(id string) (*entity.Company, error) { return validCompany(id), nil },
					hasActiveModuleFunc: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
				}
				productRepo := &fakeProductRepo{
					getByIDFunc: func(id string) (*entity.Product, error) {
						if id == testProductID1 {
							return nil, nil
						}
						return validProduct(testCompanyID, testProductID2, decimal.NewFromInt(50000), decimal.NewFromInt(5)), nil
					},
				}
				warehouseRepo := &fakeWarehouseRepo{}
				invoiceRepo := &fakeInvoiceRepo{}
				txRunner := &fakeBillingTxRunner{}
				return txRunner, &fakeInventoryUC{}, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name:      "InvalidInput_EmptyCustomerID",
			companyID: testCompanyID,
			userID:    testUserID,
			in: func() dto.CreateInvoiceRequest {
				r := validCreateInvoiceRequest()
				r.CustomerID = ""
				return r
			}(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, &fakeCustomerRepo{}, &fakeCompanyRepo{}, &fakeProductRepo{}, &fakeWarehouseRepo{}, &fakeInvoiceRepo{}
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:      "InvalidInput_EmptyItems",
			companyID: testCompanyID,
			userID:    testUserID,
			in: dto.CreateInvoiceRequest{
				CustomerID: testCustomerID,
				Prefix:     "FV",
				Items:      []dto.InvoiceItemRequest{},
			},
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, &fakeCustomerRepo{}, &fakeCompanyRepo{}, &fakeProductRepo{}, &fakeWarehouseRepo{}, &fakeInvoiceRepo{}
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:      "InvalidInput_EmptyPrefix",
			companyID: testCompanyID,
			userID:    testUserID,
			in: func() dto.CreateInvoiceRequest {
				r := validCreateInvoiceRequest()
				r.Prefix = ""
				return r
			}(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, &fakeCustomerRepo{}, &fakeCompanyRepo{}, &fakeProductRepo{}, &fakeWarehouseRepo{}, &fakeInvoiceRepo{}
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:      "InvalidInput_ItemQuantityZero",
			companyID: testCompanyID,
			userID:    testUserID,
			in: dto.CreateInvoiceRequest{
				CustomerID: testCustomerID,
				Prefix:     "FV",
				Items: []dto.InvoiceItemRequest{
					{ProductID: testProductID1, Quantity: decimal.Zero, UnitPrice: decimal.NewFromInt(10000)},
				},
			},
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{getByIDFunc: func(_ string) (*entity.Customer, error) { return validCustomer(testCompanyID), nil }}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc:         func(id string) (*entity.Company, error) { return validCompany(id), nil },
					hasActiveModuleFunc: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
				}
				productRepo := &fakeProductRepo{getByIDFunc: func(id string) (*entity.Product, error) { return validProduct(testCompanyID, id, decimal.NewFromInt(10000), decimal.NewFromInt(19)), nil }}
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, customerRepo, companyRepo, productRepo, &fakeWarehouseRepo{}, &fakeInvoiceRepo{}
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:      "InvalidInput_ItemUnitPriceNegative",
			companyID: testCompanyID,
			userID:    testUserID,
			in: dto.CreateInvoiceRequest{
				CustomerID: testCustomerID,
				Prefix:     "FV",
				Items: []dto.InvoiceItemRequest{
					{ProductID: testProductID1, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(-100)},
				},
			},
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{getByIDFunc: func(_ string) (*entity.Customer, error) { return validCustomer(testCompanyID), nil }}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc:         func(id string) (*entity.Company, error) { return validCompany(id), nil },
					hasActiveModuleFunc: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
				}
				productRepo := &fakeProductRepo{getByIDFunc: func(id string) (*entity.Product, error) { return validProduct(testCompanyID, id, decimal.NewFromInt(10000), decimal.NewFromInt(19)), nil }}
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, customerRepo, companyRepo, productRepo, &fakeWarehouseRepo{}, &fakeInvoiceRepo{}
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:      "InvalidInput_WithInventory_EmptyWarehouse",
			companyID: testCompanyID,
			userID:    testUserID,
			in:        validCreateInvoiceRequest(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{
					getByIDFunc: func(_ string) (*entity.Customer, error) { return validCustomer(testCompanyID), nil },
				}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc: func(id string) (*entity.Company, error) { return validCompany(id), nil },
					hasActiveModuleFunc: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
				}
				productRepo := &fakeProductRepo{}
				warehouseRepo := &fakeWarehouseRepo{}
				invoiceRepo := &fakeInvoiceRepo{}
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:      "InsufficientStock_WithInventory",
			companyID: testCompanyID,
			userID:    testUserID,
			in: func() dto.CreateInvoiceRequest {
				r := validCreateInvoiceRequest()
				r.WarehouseID = testWarehouseID
				return r
			}(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{
					getByIDFunc: func(_ string) (*entity.Customer, error) { return validCustomer(testCompanyID), nil },
				}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc: func(id string) (*entity.Company, error) { return validCompany(id), nil },
					hasActiveModuleFunc: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
				}
				productRepo := &fakeProductRepo{
					getByIDFunc: func(id string) (*entity.Product, error) {
						return validProduct(testCompanyID, id, decimal.NewFromInt(10000), decimal.NewFromInt(19)), nil
					},
				}
				warehouseRepo := &fakeWarehouseRepo{
					getByIDFunc: func(_ string) (*entity.Warehouse, error) { return validWarehouse(testCompanyID), nil },
				}
				invoiceRepo := &fakeInvoiceRepo{}
				inventoryUC := &fakeInventoryUC{
					registerOUTFunc: func(_ context.Context, _ repository.InventoryMovementRepository, _ repository.StockRepository, _ repository.ProductRepository, _ *entity.Product, _, _, _ string, _ decimal.Decimal, _ time.Time, _ string) error {
						return domain.ErrInsufficientStock
					},
				}
				txRunner := &fakeBillingTxRunner{
					runFunc: func(_ context.Context, fn func(
						repository.InventoryMovementRepository,
						repository.StockRepository,
						repository.ProductRepository,
						repository.CustomerRepository,
						repository.InvoiceRepository,
					) error) error {
						return fn(nil, nil, productRepo, customerRepo, invoiceRepo)
					},
				}
				return txRunner, inventoryUC, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo
			},
			wantErr: domain.ErrInsufficientStock,
		},
		{
			name:      "RepoError_InvoiceCreateFails",
			companyID: testCompanyID,
			userID:    testUserID,
			in:        validCreateInvoiceRequest(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{
					getByIDFunc: func(_ string) (*entity.Customer, error) {
						return validCustomer(testCompanyID), nil
					},
				}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc:         func(id string) (*entity.Company, error) { return validCompany(id), nil },
					hasActiveModuleFunc: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
				}
				productRepo := &fakeProductRepo{
					getByIDFunc: func(id string) (*entity.Product, error) {
						switch id {
						case testProductID1:
							return validProduct(testCompanyID, testProductID1, decimal.NewFromInt(10000), decimal.NewFromInt(19)), nil
						case testProductID2:
							return validProduct(testCompanyID, testProductID2, decimal.NewFromInt(50000), decimal.NewFromInt(5)), nil
						default:
							return nil, nil
						}
					},
				}
				warehouseRepo := &fakeWarehouseRepo{}
				invoiceRepo := &fakeInvoiceRepo{
					createFunc: func(_ *entity.Invoice) error {
						return errors.New("db: constraint violation")
					},
					createDetailFunc: func(_ *entity.InvoiceDetail) error { return nil },
				}
				txRunner := &fakeBillingTxRunner{
					runFunc: func(_ context.Context, fn func(
						repository.InventoryMovementRepository,
						repository.StockRepository,
						repository.ProductRepository,
						repository.CustomerRepository,
						repository.InvoiceRepository,
					) error) error {
						return fn(nil, nil, productRepo, customerRepo, invoiceRepo)
					},
				}
				return txRunner, &fakeInventoryUC{}, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txRunner, inventoryUC, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo := tt.setup()
			uc := NewCreateInvoiceUseCase(
				txRunner,
				inventoryUC,
				customerRepo,
				companyRepo,
				productRepo,
				warehouseRepo,
				invoiceRepo,
				nil,
				dianConfig,
			)

			out, err := uc.CreateInvoice(ctx, tt.companyID, tt.userID, tt.in)

			if tt.wantAnyErr {
				require.Error(t, err)
				assert.Nil(t, out)
				return
			}
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, out)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, out)
			if tt.validateOut != nil {
				tt.validateOut(t, out)
			}
		})
	}
}
