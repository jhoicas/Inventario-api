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

// testInvoiceID factura original a la que se aplica la nota de crédito en los tests.
const testInvoiceID = "invoice-original-001"

// ── Helpers de dominio para Nota Crédito ───────────────────────────────────────

// validOriginalInvoice devuelve una factura original de prueba (vendido 2×product1, 1×product2).
func validOriginalInvoice(companyID, invoiceID string) *entity.Invoice {
	now := time.Now()
	return &entity.Invoice{
		ID:          invoiceID,
		CompanyID:   companyID,
		CustomerID:  testCustomerID,
		Prefix:      "FV",
		Number:      "FV-1001",
		Date:        now,
		NetTotal:    decimal.NewFromInt(70000),
		TaxTotal:    decimal.NewFromInt(6300),
		GrandTotal:  decimal.NewFromInt(76300),
		DIAN_Status: entity.DIANStatusExitoso,
		CUFE:        "cufe-original-123",
		DocumentType: "INVOICE",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// validOriginalDetails devuelve los detalles de la factura original: 2 ud product1 @ 10000, 1 ud product2 @ 50000.
func validOriginalDetails(invoiceID string) []*entity.InvoiceDetail {
	return []*entity.InvoiceDetail{
		{ID: "det-1", InvoiceID: invoiceID, ProductID: testProductID1, Quantity: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(10000), TaxRate: decimal.NewFromFloat(0.19), Subtotal: decimal.NewFromInt(20000)},
		{ID: "det-2", InvoiceID: invoiceID, ProductID: testProductID2, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50000), TaxRate: decimal.NewFromFloat(0.05), Subtotal: decimal.NewFromInt(50000)},
	}
}

// validCreateCreditNoteRequest devolución parcial: 1 ud de product1 y 1 ud de product2.
func validCreateCreditNoteRequest() dto.ReturnInvoiceRequest {
	return dto.ReturnInvoiceRequest{
		WarehouseID: testWarehouseID,
		Items: []dto.ReturnItemRequest{
			{ProductID: testProductID1, Quantity: decimal.NewFromInt(1)},
			{ProductID: testProductID2, Quantity: decimal.NewFromInt(1)},
		},
		Reason: "Devolución parcial por defecto",
	}
}

// ── Tests CreateCreditNote (Table-Driven) ───────────────────────────────────────

func TestCreateCreditNoteUseCase_CreateCreditNote(t *testing.T) {
	ctx := context.Background()
	dianConfig := DIANConfig{TechnicalKey: ""}

	tests := []struct {
		name        string
		companyID   string
		userID      string
		invoiceID   string
		in          dto.ReturnInvoiceRequest
		setup       func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo)
		wantErr     error
		wantAnyErr  bool
		validateOut func(t *testing.T, out *dto.InvoiceResponse)
	}{
		{
			name:      "Success_NoInventory_ReversiónCompleta",
			companyID: testCompanyID,
			userID:    testUserID,
			invoiceID: testInvoiceID,
			in:        validCreateCreditNoteRequest(),
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
					getByIDFunc: func(id string) (*entity.Company, error) { return validCompany(id), nil },
					hasActiveModuleFunc: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
				}
				productRepo := &fakeProductRepo{}
				warehouseRepo := &fakeWarehouseRepo{}
				invoiceRepo := &fakeInvoiceRepo{
					getByIDFunc: func(id string) (*entity.Invoice, error) {
						if id != testInvoiceID {
							return nil, nil
						}
						return validOriginalInvoice(testCompanyID, testInvoiceID), nil
					},
					getDetailsByInvoiceIDFunc: func(id string) ([]*entity.InvoiceDetail, error) {
						return validOriginalDetails(id), nil
					},
					createFunc:       func(inv *entity.Invoice) error { return nil },
					createDetailFunc: func(_ *entity.InvoiceDetail) error { return nil },
					updateReturnStatusFunc: func(invoiceID string, status string) error {
						assert.Equal(t, testInvoiceID, invoiceID)
						assert.Contains(t, []string{"Returned", "Partially_Returned"}, status)
						return nil
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
				return txRunner, &fakeInventoryUC{}, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo
			},
			wantErr: nil,
			validateOut: func(t *testing.T, out *dto.InvoiceResponse) {
				require.NotNil(t, out)
				assert.Equal(t, entity.DIANStatusDraft, out.DIAN_Status)
				require.Len(t, out.Details, 2)
				// Devolución 1×10000 + 1×50000 = 60000 neto; tax 1900+2500 = 4400; total 64400
				expectedNet := decimal.NewFromInt(60000)
				expectedTax := decimal.NewFromInt(1900).Add(decimal.NewFromInt(2500))
				expectedGrand := expectedNet.Add(expectedTax)
				assert.True(t, out.NetTotal.Equal(expectedNet), "NetTotal")
				assert.True(t, out.TaxTotal.Equal(expectedTax), "TaxTotal")
				assert.True(t, out.GrandTotal.Equal(expectedGrand), "GrandTotal")
				assert.Equal(t, "Cliente Prueba", out.CustomerName)
			},
		},
		{
			name:      "Success_WithInventory_RegisterReturnInTxInvocado",
			companyID: testCompanyID,
			userID:    testUserID,
			invoiceID: testInvoiceID,
			in:        validCreateCreditNoteRequest(),
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
				invoiceRepo := &fakeInvoiceRepo{
					getByIDFunc: func(id string) (*entity.Invoice, error) {
						if id != testInvoiceID { return nil, nil }
						return validOriginalInvoice(testCompanyID, testInvoiceID), nil
					},
					getDetailsByInvoiceIDFunc: func(id string) ([]*entity.InvoiceDetail, error) { return validOriginalDetails(id), nil },
					createFunc:                func(_ *entity.Invoice) error { return nil },
					createDetailFunc:          func(_ *entity.InvoiceDetail) error { return nil },
					updateReturnStatusFunc:    func(_, _ string) error { return nil },
				}
				registerReturnCalls := 0
				inventoryUC := &fakeInventoryUC{
					registerReturnFunc: func(_ context.Context, _ repository.InventoryMovementRepository, _ repository.StockRepository, _ repository.ProductRepository, _ *entity.Product, productID, warehouseID, _ string, qty decimal.Decimal, _ time.Time, txID string) error {
						registerReturnCalls++
						assert.Equal(t, testWarehouseID, warehouseID)
						assert.NotEmpty(t, txID)
						assert.True(t, qty.GreaterThan(decimal.Zero))
						assert.Contains(t, []string{testProductID1, testProductID2}, productID)
						return nil
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
						err := fn(nil, nil, productRepo, customerRepo, invoiceRepo)
						// CRÍTICO: dentro de la tx se deben haber llamado RegisterReturnInTx por cada ítem (2 ítems).
						assert.Equal(t, 2, registerReturnCalls, "RegisterReturnInTx debe invocarse una vez por cada ítem devuelto")
						return err
					},
				}
				return txRunner, inventoryUC, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo
			},
			wantErr: nil,
			validateOut: func(t *testing.T, out *dto.InvoiceResponse) {
				require.NotNil(t, out)
				require.Len(t, out.Details, 2)
			},
		},
		{
			name:      "InvoiceNotFound",
			companyID: testCompanyID,
			userID:    testUserID,
			invoiceID: testInvoiceID,
			in:        validCreateCreditNoteRequest(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				invoiceRepo := &fakeInvoiceRepo{
					getByIDFunc: func(_ string) (*entity.Invoice, error) { return nil, nil },
				}
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, &fakeCustomerRepo{}, &fakeCompanyRepo{}, &fakeProductRepo{}, &fakeWarehouseRepo{}, invoiceRepo
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name:      "Forbidden_InvoiceFromOtherCompany",
			companyID: testCompanyID,
			userID:    testUserID,
			invoiceID: testInvoiceID,
			in:        validCreateCreditNoteRequest(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				invoiceRepo := &fakeInvoiceRepo{
					getByIDFunc: func(_ string) (*entity.Invoice, error) {
						return validOriginalInvoice("otra-empresa", testInvoiceID), nil
					},
				}
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, &fakeCustomerRepo{}, &fakeCompanyRepo{}, &fakeProductRepo{}, &fakeWarehouseRepo{}, invoiceRepo
			},
			wantErr: domain.ErrForbidden,
		},
		{
			name:      "InvalidInput_ReturnMoreThanSold",
			companyID: testCompanyID,
			userID:    testUserID,
			invoiceID: testInvoiceID,
			in: dto.ReturnInvoiceRequest{
				Items: []dto.ReturnItemRequest{
					{ProductID: testProductID1, Quantity: decimal.NewFromInt(10)}, // vendidos 2, devolvemos 10
				},
			},
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{getByIDFunc: func(_ string) (*entity.Customer, error) { return validCustomer(testCompanyID), nil }}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc:         func(id string) (*entity.Company, error) { return validCompany(id), nil },
					hasActiveModuleFunc: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
				}
				invoiceRepo := &fakeInvoiceRepo{
					getByIDFunc: func(id string) (*entity.Invoice, error) {
						if id != testInvoiceID { return nil, nil }
						return validOriginalInvoice(testCompanyID, testInvoiceID), nil
					},
					getDetailsByInvoiceIDFunc: func(id string) ([]*entity.InvoiceDetail, error) { return validOriginalDetails(id), nil },
				}
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, customerRepo, companyRepo, &fakeProductRepo{}, &fakeWarehouseRepo{}, invoiceRepo
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:      "InvalidInput_ProductNotInOriginalInvoice",
			companyID: testCompanyID,
			userID:    testUserID,
			invoiceID: testInvoiceID,
			in: dto.ReturnInvoiceRequest{
				Items: []dto.ReturnItemRequest{
					{ProductID: "product-no-vendido", Quantity: decimal.NewFromInt(1)},
				},
			},
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{getByIDFunc: func(_ string) (*entity.Customer, error) { return validCustomer(testCompanyID), nil }}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc:         func(id string) (*entity.Company, error) { return validCompany(id), nil },
					hasActiveModuleFunc: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
				}
				invoiceRepo := &fakeInvoiceRepo{
					getByIDFunc: func(id string) (*entity.Invoice, error) {
						if id != testInvoiceID { return nil, nil }
						return validOriginalInvoice(testCompanyID, testInvoiceID), nil
					},
					getDetailsByInvoiceIDFunc: func(id string) ([]*entity.InvoiceDetail, error) { return validOriginalDetails(id), nil },
				}
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, customerRepo, companyRepo, &fakeProductRepo{}, &fakeWarehouseRepo{}, invoiceRepo
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:      "InvalidInput_EmptyInvoiceID",
			companyID: testCompanyID,
			userID:    testUserID,
			invoiceID: "",
			in:        validCreateCreditNoteRequest(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, &fakeCustomerRepo{}, &fakeCompanyRepo{}, &fakeProductRepo{}, &fakeWarehouseRepo{}, &fakeInvoiceRepo{}
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:      "InvalidInput_EmptyItems",
			companyID: testCompanyID,
			userID:    testUserID,
			invoiceID: testInvoiceID,
			in:        dto.ReturnInvoiceRequest{Items: nil},
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, &fakeCustomerRepo{}, &fakeCompanyRepo{}, &fakeProductRepo{}, &fakeWarehouseRepo{}, &fakeInvoiceRepo{}
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:      "InvalidInput_WithInventory_EmptyWarehouse",
			companyID: testCompanyID,
			userID:    testUserID,
			invoiceID: testInvoiceID,
			in: dto.ReturnInvoiceRequest{
				WarehouseID: "",
				Items:       validCreateCreditNoteRequest().Items,
				Reason:      "test",
			},
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{getByIDFunc: func(_ string) (*entity.Customer, error) { return validCustomer(testCompanyID), nil }}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc:         func(id string) (*entity.Company, error) { return validCompany(id), nil },
					hasActiveModuleFunc: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
				}
				invoiceRepo := &fakeInvoiceRepo{
					getByIDFunc: func(id string) (*entity.Invoice, error) {
						if id != testInvoiceID { return nil, nil }
						return validOriginalInvoice(testCompanyID, testInvoiceID), nil
					},
					getDetailsByInvoiceIDFunc: func(id string) ([]*entity.InvoiceDetail, error) { return validOriginalDetails(id), nil },
				}
				return &fakeBillingTxRunner{}, &fakeInventoryUC{}, customerRepo, companyRepo, &fakeProductRepo{}, &fakeWarehouseRepo{}, invoiceRepo
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:      "RepoError_InvoiceCreateFails",
			companyID: testCompanyID,
			userID:    testUserID,
			invoiceID: testInvoiceID,
			in:        validCreateCreditNoteRequest(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{getByIDFunc: func(_ string) (*entity.Customer, error) { return validCustomer(testCompanyID), nil }}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc:         func(id string) (*entity.Company, error) { return validCompany(id), nil },
					hasActiveModuleFunc: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
				}
				invoiceRepo := &fakeInvoiceRepo{
					getByIDFunc: func(id string) (*entity.Invoice, error) {
						if id != testInvoiceID { return nil, nil }
						return validOriginalInvoice(testCompanyID, testInvoiceID), nil
					},
					getDetailsByInvoiceIDFunc: func(id string) ([]*entity.InvoiceDetail, error) { return validOriginalDetails(id), nil },
					createFunc: func(_ *entity.Invoice) error {
						return errors.New("db: constraint violation")
					},
					createDetailFunc:       func(_ *entity.InvoiceDetail) error { return nil },
					updateReturnStatusFunc: func(_, _ string) error { return nil },
				}
				txRunner := &fakeBillingTxRunner{
					runFunc: func(_ context.Context, fn func(
						repository.InventoryMovementRepository,
						repository.StockRepository,
						repository.ProductRepository,
						repository.CustomerRepository,
						repository.InvoiceRepository,
					) error) error {
						return fn(nil, nil, &fakeProductRepo{}, customerRepo, invoiceRepo)
					},
				}
				return txRunner, &fakeInventoryUC{}, customerRepo, companyRepo, &fakeProductRepo{}, &fakeWarehouseRepo{}, invoiceRepo
			},
			wantAnyErr: true,
		},
		{
			name:      "RepoError_UpdateReturnStatusFails",
			companyID: testCompanyID,
			userID:    testUserID,
			invoiceID: testInvoiceID,
			in:        validCreateCreditNoteRequest(),
			setup: func() (*fakeBillingTxRunner, *fakeInventoryUC, *fakeCustomerRepo, *fakeCompanyRepo, *fakeProductRepo, *fakeWarehouseRepo, *fakeInvoiceRepo) {
				customerRepo := &fakeCustomerRepo{getByIDFunc: func(_ string) (*entity.Customer, error) { return validCustomer(testCompanyID), nil }}
				companyRepo := &fakeCompanyRepo{
					getByIDFunc:         func(id string) (*entity.Company, error) { return validCompany(id), nil },
					hasActiveModuleFunc: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
				}
				invoiceRepo := &fakeInvoiceRepo{
					getByIDFunc: func(id string) (*entity.Invoice, error) {
						if id != testInvoiceID { return nil, nil }
						return validOriginalInvoice(testCompanyID, testInvoiceID), nil
					},
					getDetailsByInvoiceIDFunc: func(id string) ([]*entity.InvoiceDetail, error) { return validOriginalDetails(id), nil },
					createFunc:               func(_ *entity.Invoice) error { return nil },
					createDetailFunc:         func(_ *entity.InvoiceDetail) error { return nil },
					updateReturnStatusFunc: func(_, _ string) error {
						return errors.New("db: update return status failed")
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
						return fn(nil, nil, &fakeProductRepo{}, customerRepo, invoiceRepo)
					},
				}
				return txRunner, &fakeInventoryUC{}, customerRepo, companyRepo, &fakeProductRepo{}, &fakeWarehouseRepo{}, invoiceRepo
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txRunner, inventoryUC, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo := tt.setup()
			uc := NewCreateCreditNoteUseCase(
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

			out, err := uc.CreateCreditNote(ctx, tt.companyID, tt.userID, tt.invoiceID, tt.in)

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
