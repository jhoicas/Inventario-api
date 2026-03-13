package postgres

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/shopspring/decimal"
)

type poTxFake struct {
	execSQLs  []string
	execArgs  [][]any
	execErr   error
	queryRows pgx.Rows
	queryErr  error
	queryRow  pgx.Row
	commits   int
	rollbacks int
	cmdTag    pgconn.CommandTag
}

func (f *poTxFake) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.execSQLs = append(f.execSQLs, sql)
	f.execArgs = append(f.execArgs, args)
	if f.execErr != nil {
		return pgconn.CommandTag{}, f.execErr
	}
	if f.cmdTag != (pgconn.CommandTag{}) {
		return f.cmdTag, nil
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (f *poTxFake) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	if f.queryRows != nil {
		return f.queryRows, nil
	}
	return &poRowsFake{}, nil
}

func (f *poTxFake) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	if f.queryRow != nil {
		return f.queryRow
	}
	return &poRowFake{err: pgx.ErrNoRows}
}

func (f *poTxFake) Commit(_ context.Context) error {
	f.commits++
	return nil
}

func (f *poTxFake) Rollback(_ context.Context) error {
	f.rollbacks++
	return nil
}

type poRowFake struct {
	values []any
	err    error
}

func (r *poRowFake) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("scan destination size mismatch")
	}
	for i := range dest {
		dv := reflect.ValueOf(dest[i])
		if dv.Kind() != reflect.Ptr || dv.IsNil() {
			return errors.New("scan destination must be non-nil pointer")
		}
		sv := reflect.ValueOf(r.values[i])
		dv.Elem().Set(sv)
	}
	return nil
}

type poRowsFake struct {
	rows   [][]any
	idx    int
	closed bool
	err    error
}

func (r *poRowsFake) Close() {
	r.closed = true
}

func (r *poRowsFake) Err() error {
	return r.err
}

func (r *poRowsFake) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 1")
}

func (r *poRowsFake) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *poRowsFake) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}

func (r *poRowsFake) Scan(dest ...any) error {
	if r.idx == 0 || r.idx > len(r.rows) {
		return errors.New("scan called without current row")
	}
	vals := r.rows[r.idx-1]
	if len(dest) != len(vals) {
		return errors.New("scan destination size mismatch")
	}
	for i := range dest {
		dv := reflect.ValueOf(dest[i])
		if dv.Kind() != reflect.Ptr || dv.IsNil() {
			return errors.New("scan destination must be non-nil pointer")
		}
		sv := reflect.ValueOf(vals[i])
		dv.Elem().Set(sv)
	}
	return nil
}

func (r *poRowsFake) Values() ([]any, error) {
	if r.idx == 0 || r.idx > len(r.rows) {
		return nil, errors.New("no current row")
	}
	return r.rows[r.idx-1], nil
}

func (r *poRowsFake) RawValues() [][]byte {
	return nil
}

func (r *poRowsFake) Conn() *pgx.Conn {
	return nil
}

func purchaseOrderFixture() *entity.PurchaseOrder {
	now := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	return &entity.PurchaseOrder{
		ID:         "po-1",
		CompanyID:  "co-1",
		SupplierID: "sup-1",
		Number:     "PO-0001",
		Date:       now,
		Status:     entity.PurchaseOrderStatusDraft,
		Items: []entity.PurchaseOrderItem{
			{ProductID: "prod-1", Quantity: decimal.RequireFromString("2.5"), UnitCost: decimal.RequireFromString("100.10")},
			{ProductID: "prod-2", Quantity: decimal.RequireFromString("1"), UnitCost: decimal.RequireFromString("20")},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestPurchaseOrderRepo_Create_InsertsHeaderAndItems(t *testing.T) {
	txFake := &poTxFake{}
	repo := NewPurchaseOrderRepository(txFake)

	po := purchaseOrderFixture()
	if err := repo.Create(context.Background(), po); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txFake.execSQLs) != 3 {
		t.Fatalf("expected 3 exec calls (1 header + 2 items), got %d", len(txFake.execSQLs))
	}
	if !strings.Contains(txFake.execSQLs[0], "INSERT INTO purchase_orders") {
		t.Fatalf("expected first insert into purchase_orders, got: %s", txFake.execSQLs[0])
	}
	if !strings.Contains(txFake.execSQLs[1], "INSERT INTO purchase_order_items") || !strings.Contains(txFake.execSQLs[2], "INSERT INTO purchase_order_items") {
		t.Fatalf("expected item inserts into purchase_order_items")
	}
	if txFake.commits != 0 {
		t.Fatalf("expected 0 commits for existing tx path, got %d", txFake.commits)
	}
	if txFake.rollbacks != 0 {
		t.Fatalf("expected 0 rollbacks, got %d", txFake.rollbacks)
	}
}

func TestPurchaseOrderRepo_Create_MapsUniqueViolationToErrDuplicate(t *testing.T) {
	txFake := &poTxFake{execErr: &pgconn.PgError{Code: "23505"}}
	repo := NewPurchaseOrderRepository(txFake)

	err := repo.Create(context.Background(), purchaseOrderFixture())
	if !errors.Is(err, domain.ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestPurchaseOrderRepo_UpdateStatus_OK(t *testing.T) {
	txFake := &poTxFake{cmdTag: pgconn.NewCommandTag("UPDATE 1")}
	repo := NewPurchaseOrderRepository(txFake)

	err := repo.UpdateStatus(context.Background(), "po-1", entity.PurchaseOrderStatusClosed, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txFake.execSQLs) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(txFake.execSQLs))
	}
	if !strings.Contains(txFake.execSQLs[0], "UPDATE purchase_orders") {
		t.Fatalf("expected UPDATE purchase_orders SQL, got: %s", txFake.execSQLs[0])
	}
}

func TestPurchaseOrderRepo_UpdateStatus_NotFound(t *testing.T) {
	txFake := &poTxFake{cmdTag: pgconn.NewCommandTag("UPDATE 0")}
	repo := NewPurchaseOrderRepository(txFake)

	err := repo.UpdateStatus(context.Background(), "po-missing", entity.PurchaseOrderStatusClosed, time.Now())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPurchaseOrderRepo_GetByID_FoundWithItems(t *testing.T) {
	now := time.Date(2026, 3, 12, 10, 30, 0, 0, time.UTC)
	txFake := &poTxFake{
		queryRow: &poRowFake{values: []any{
			"po-1",
			"co-1",
			"sup-1",
			"PO-0001",
			now,
			entity.PurchaseOrderStatusDraft,
			now,
			now,
		}},
		queryRows: &poRowsFake{rows: [][]any{
			{"prod-1", decimal.RequireFromString("2.5"), decimal.RequireFromString("100.10")},
			{"prod-2", decimal.RequireFromString("1"), decimal.RequireFromString("20")},
		}},
	}
	repo := NewPurchaseOrderRepository(txFake)

	po, err := repo.GetByID(context.Background(), "po-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if po == nil {
		t.Fatal("expected purchase order, got nil")
	}
	if po.ID != "po-1" || po.CompanyID != "co-1" || po.SupplierID != "sup-1" {
		t.Fatalf("unexpected purchase order header: %+v", po)
	}
	if len(po.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(po.Items))
	}
	if po.Items[0].ProductID != "prod-1" || !po.Items[0].Quantity.Equal(decimal.RequireFromString("2.5")) {
		t.Fatalf("unexpected first item: %+v", po.Items[0])
	}
}

func TestPurchaseOrderRepo_GetByID_NotFound(t *testing.T) {
	txFake := &poTxFake{queryRow: &poRowFake{err: pgx.ErrNoRows}}
	repo := NewPurchaseOrderRepository(txFake)

	po, err := repo.GetByID(context.Background(), "po-missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if po != nil {
		t.Fatalf("expected nil purchase order, got %+v", po)
	}
}

func TestPurchaseOrderRepo_GetByID_ItemsQueryError(t *testing.T) {
	now := time.Date(2026, 3, 12, 10, 30, 0, 0, time.UTC)
	txFake := &poTxFake{
		queryRow: &poRowFake{values: []any{
			"po-1",
			"co-1",
			"sup-1",
			"PO-0001",
			now,
			entity.PurchaseOrderStatusDraft,
			now,
			now,
		}},
		queryErr: errors.New("query failed"),
	}
	repo := NewPurchaseOrderRepository(txFake)

	po, err := repo.GetByID(context.Background(), "po-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if po != nil {
		t.Fatalf("expected nil purchase order on error, got %+v", po)
	}
	if !strings.Contains(err.Error(), "list purchase order items") {
		t.Fatalf("expected wrapped items query context, got %v", err)
	}
}

func TestPurchaseOrderRepo_GetByID_ItemsRowsErr(t *testing.T) {
	now := time.Date(2026, 3, 12, 10, 30, 0, 0, time.UTC)
	txFake := &poTxFake{
		queryRow: &poRowFake{values: []any{
			"po-1",
			"co-1",
			"sup-1",
			"PO-0001",
			now,
			entity.PurchaseOrderStatusDraft,
			now,
			now,
		}},
		queryRows: &poRowsFake{err: errors.New("rows iteration failed")},
	}
	repo := NewPurchaseOrderRepository(txFake)

	po, err := repo.GetByID(context.Background(), "po-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if po != nil {
		t.Fatalf("expected nil purchase order on error, got %+v", po)
	}
	if !strings.Contains(err.Error(), "iterate purchase order items") {
		t.Fatalf("expected wrapped rows iteration context, got %v", err)
	}
}

func TestPurchaseOrderRepo_GetByID_ItemScanError(t *testing.T) {
	now := time.Date(2026, 3, 12, 10, 30, 0, 0, time.UTC)
	txFake := &poTxFake{
		queryRow: &poRowFake{values: []any{
			"po-1",
			"co-1",
			"sup-1",
			"PO-0001",
			now,
			entity.PurchaseOrderStatusDraft,
			now,
			now,
		}},
		queryRows: &poRowsFake{rows: [][]any{
			{"prod-1", decimal.RequireFromString("2.5")},
		}},
	}
	repo := NewPurchaseOrderRepository(txFake)

	po, err := repo.GetByID(context.Background(), "po-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if po != nil {
		t.Fatalf("expected nil purchase order on error, got %+v", po)
	}
	if !strings.Contains(err.Error(), "scan purchase order item") {
		t.Fatalf("expected wrapped scan context, got %v", err)
	}
}
