package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jhoicas/Inventario-api/internal/application/billing"
	"github.com/jhoicas/Inventario-api/internal/application/inventory"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// Ensure TxRunner implements inventory.TxRunner and billing.BillingTxRunner.
var _ inventory.TxRunner = (*TxRunner)(nil)
var _ billing.BillingTxRunner = (*TxRunner)(nil)

// TxRunner ejecuta callbacks dentro de una transacción PostgreSQL.
type TxRunner struct {
	pool *pgxpool.Pool
}

// NewTxRunner construye el runner con el pool.
func NewTxRunner(pool *pgxpool.Pool) *TxRunner {
	return &TxRunner{pool: pool}
}

// Run inicia una transacción, ejecuta fn con repos atados a la tx y hace Commit o Rollback.
func (r *TxRunner) Run(ctx context.Context, fn func(
	movRepo repository.InventoryMovementRepository,
	stockRepo repository.StockRepository,
	productRepo repository.ProductRepository,
) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	movRepo := NewInventoryMovementRepository(tx)
	stockRepo := NewStockRepository(tx)
	productRepo := NewProductRepository(tx)

	if err := fn(movRepo, stockRepo, productRepo); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// RunBilling inicia una transacción con repos de inventario y facturación (para CreateInvoice).
func (r *TxRunner) RunBilling(ctx context.Context, fn func(
	movRepo repository.InventoryMovementRepository,
	stockRepo repository.StockRepository,
	productRepo repository.ProductRepository,
	customerRepo repository.CustomerRepository,
	invoiceRepo repository.InvoiceRepository,
) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	movRepo := NewInventoryMovementRepository(tx)
	stockRepo := NewStockRepository(tx)
	productRepo := NewProductRepository(tx)
	customerRepo := NewCustomerRepository(tx)
	invoiceRepo := NewInvoiceRepository(tx)

	if err := fn(movRepo, stockRepo, productRepo, customerRepo, invoiceRepo); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
