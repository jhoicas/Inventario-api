package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/shopspring/decimal"
)

type reorderConfigQuerierFake struct {
	execSQL  string
	execArgs []any
	execErr  error
}

func (f *reorderConfigQuerierFake) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.execSQL = sql
	f.execArgs = args
	return pgconn.CommandTag{}, f.execErr
}

func (f *reorderConfigQuerierFake) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, nil
}

func (f *reorderConfigQuerierFake) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return nil
}

func TestReorderConfigRepo_UpsertProductReorderConfig_OK(t *testing.T) {
	fake := &reorderConfigQuerierFake{}
	repo := NewReorderConfigRepository(fake)

	in := dto.ReorderConfigRequest{
		ProductID:    "prod-1",
		WarehouseID:  "wh-1",
		ReorderPoint: decimal.RequireFromString("12.5"),
		MinStock:     decimal.RequireFromString("5"),
		MaxStock:     decimal.RequireFromString("40"),
		LeadTimeDays: 7,
	}

	err := repo.UpsertProductReorderConfig(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(fake.execSQL, "INSERT INTO product_reorder_config") {
		t.Fatalf("expected INSERT INTO product_reorder_config, got SQL: %s", fake.execSQL)
	}
	if !strings.Contains(fake.execSQL, "ON CONFLICT (product_id, warehouse_id)") {
		t.Fatalf("expected ON CONFLICT clause, got SQL: %s", fake.execSQL)
	}

	if len(fake.execArgs) != 6 {
		t.Fatalf("expected 6 args, got %d", len(fake.execArgs))
	}
	if got := fake.execArgs[0]; got != in.ProductID {
		t.Fatalf("arg1 product_id mismatch: got %v want %v", got, in.ProductID)
	}
	if got := fake.execArgs[1]; got != in.WarehouseID {
		t.Fatalf("arg2 warehouse_id mismatch: got %v want %v", got, in.WarehouseID)
	}
	if got := fake.execArgs[2].(decimal.Decimal); !got.Equal(in.ReorderPoint) {
		t.Fatalf("arg3 reorder_point mismatch: got %s want %s", got.String(), in.ReorderPoint.String())
	}
	if got := fake.execArgs[3].(decimal.Decimal); !got.Equal(in.MinStock) {
		t.Fatalf("arg4 min_stock mismatch: got %s want %s", got.String(), in.MinStock.String())
	}
	if got := fake.execArgs[4].(decimal.Decimal); !got.Equal(in.MaxStock) {
		t.Fatalf("arg5 max_stock mismatch: got %s want %s", got.String(), in.MaxStock.String())
	}
	if got := fake.execArgs[5]; got != in.LeadTimeDays {
		t.Fatalf("arg6 lead_time_days mismatch: got %v want %v", got, in.LeadTimeDays)
	}
}

func TestReorderConfigRepo_UpsertProductReorderConfig_DBError(t *testing.T) {
	expectedErr := errors.New("db down")
	fake := &reorderConfigQuerierFake{execErr: expectedErr}
	repo := NewReorderConfigRepository(fake)

	err := repo.UpsertProductReorderConfig(context.Background(), dto.ReorderConfigRequest{
		ProductID:    "prod-1",
		WarehouseID:  "wh-1",
		ReorderPoint: decimal.NewFromInt(10),
		MinStock:     decimal.NewFromInt(2),
		MaxStock:     decimal.NewFromInt(20),
		LeadTimeDays: 3,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "upsert product reorder config") {
		t.Fatalf("expected wrapped error context, got: %v", err)
	}
}
