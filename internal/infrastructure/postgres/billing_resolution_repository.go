package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

var _ repository.BillingResolutionRepository = (*BillingResolutionRepo)(nil)

// BillingResolutionRepo implementa BillingResolutionRepository sobre PostgreSQL.
type BillingResolutionRepo struct {
	pool *pgxpool.Pool
}

// NewBillingResolutionRepository construye el repositorio.
func NewBillingResolutionRepository(pool *pgxpool.Pool) *BillingResolutionRepo {
	return &BillingResolutionRepo{pool: pool}
}

func (r *BillingResolutionRepo) Create(ctx context.Context, res *entity.BillingResolution) error {
	const q = `
		INSERT INTO billing_resolutions
			(id, company_id, resolution_number, prefix, range_from, range_to, date_from, date_to, is_active, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now())`
	_, err := r.pool.Exec(ctx, q,
		res.ID, res.CompanyID, res.ResolutionNumber, res.Prefix,
		res.RangeFrom, res.RangeTo, res.DateFrom, res.DateTo, res.IsActive,
	)
	if err != nil {
		return fmt.Errorf("insert billing_resolution: %w", err)
	}
	return nil
}

func (r *BillingResolutionRepo) GetByID(ctx context.Context, id string) (*entity.BillingResolution, error) {
	const q = `
		SELECT id, company_id, resolution_number, prefix, range_from, range_to,
		       date_from, date_to, is_active, created_at, updated_at
		FROM billing_resolutions WHERE id = $1`
	res, err := scanResolution(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get billing_resolution by id: %w", err)
	}
	return res, nil
}

// GetActiveByCompanyAndPrefix es la consulta crítica del flujo DIAN.
// Devuelve nil, nil si no hay resolución activa (la factura se genera sin DianExtensions).
func (r *BillingResolutionRepo) GetActiveByCompanyAndPrefix(ctx context.Context, companyID, prefix string) (*entity.BillingResolution, error) {
	const q = `
		SELECT id, company_id, resolution_number, prefix, range_from, range_to,
		       date_from, date_to, is_active, created_at, updated_at
		FROM billing_resolutions
		WHERE company_id = $1
		  AND prefix     = $2
		  AND is_active  = true
		  AND date_to   >= CURRENT_DATE
		ORDER BY date_from DESC
		LIMIT 1`
	res, err := scanResolution(r.pool.QueryRow(ctx, q, companyID, prefix))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // no hay resolución activa: factura sin DianExtensions
		}
		return nil, fmt.Errorf("get active billing_resolution: %w", err)
	}
	return res, nil
}

func (r *BillingResolutionRepo) ListByCompany(ctx context.Context, companyID string) ([]*entity.BillingResolution, error) {
	const q = `
		SELECT id, company_id, resolution_number, prefix, range_from, range_to,
		       date_from, date_to, is_active, created_at, updated_at
		FROM billing_resolutions
		WHERE company_id = $1
		ORDER BY date_from DESC`
	rows, err := r.pool.Query(ctx, q, companyID)
	if err != nil {
		return nil, fmt.Errorf("list billing_resolutions: %w", err)
	}
	defer rows.Close()
	var list []*entity.BillingResolution
	for rows.Next() {
		res, err := scanResolution(rows)
		if err != nil {
			return nil, fmt.Errorf("scan billing_resolution: %w", err)
		}
		list = append(list, res)
	}
	return list, rows.Err()
}

func (r *BillingResolutionRepo) Update(ctx context.Context, res *entity.BillingResolution) error {
	const q = `
		UPDATE billing_resolutions
		SET resolution_number = $2, prefix = $3, range_from = $4, range_to = $5,
		    date_from = $6, date_to = $7, is_active = $8, updated_at = now()
		WHERE id = $1`
	_, err := r.pool.Exec(ctx, q,
		res.ID, res.ResolutionNumber, res.Prefix,
		res.RangeFrom, res.RangeTo, res.DateFrom, res.DateTo, res.IsActive,
	)
	if err != nil {
		return fmt.Errorf("update billing_resolution: %w", err)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// pgxScanner abstrae pgx.Row y pgx.Rows para reutilizar scanResolution.
type pgxScanner interface {
	Scan(dest ...any) error
}

func scanResolution(row pgxScanner) (*entity.BillingResolution, error) {
	var res entity.BillingResolution
	err := row.Scan(
		&res.ID, &res.CompanyID, &res.ResolutionNumber, &res.Prefix,
		&res.RangeFrom, &res.RangeTo,
		&res.DateFrom, &res.DateTo,
		&res.IsActive, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
