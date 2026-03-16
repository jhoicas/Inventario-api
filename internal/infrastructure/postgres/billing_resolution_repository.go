package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	const qWithIsActive = `
		INSERT INTO billing_resolutions
			(id, company_id, resolution_number, prefix, range_from, range_to, date_from, date_to, environment, is_active, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), now())`
	_, err := r.pool.Exec(ctx, qWithIsActive,
		res.ID, res.CompanyID, res.ResolutionNumber, res.Prefix,
		res.RangeFrom, res.RangeTo, res.DateFrom, res.DateTo, res.Environment, res.IsActive,
	)
	if isUndefinedColumnError(err, "is_active") {
		const qWithoutIsActive = `
			INSERT INTO billing_resolutions
				(id, company_id, resolution_number, prefix, range_from, range_to, date_from, date_to, environment, created_at, updated_at)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now())`
		_, err = r.pool.Exec(ctx, qWithoutIsActive,
			res.ID, res.CompanyID, res.ResolutionNumber, res.Prefix,
			res.RangeFrom, res.RangeTo, res.DateFrom, res.DateTo, res.Environment,
		)
	}
	if err != nil {
		return fmt.Errorf("insert billing_resolution: %w", err)
	}
	return nil
}

func (r *BillingResolutionRepo) GetByID(ctx context.Context, id string) (*entity.BillingResolution, error) {
	const qWithIsActive = `
		SELECT id, company_id, resolution_number, prefix, range_from, range_to,
		       date_from, date_to, environment, 0::bigint AS used_numbers, is_active, created_at, updated_at
		FROM billing_resolutions WHERE id = $1`
	res, err := scanResolution(r.pool.QueryRow(ctx, qWithIsActive, id))
	if isUndefinedColumnError(err, "is_active") {
		const qWithoutIsActive = `
			SELECT id, company_id, resolution_number, prefix, range_from, range_to,
			       date_from, date_to, environment, 0::bigint AS used_numbers, true AS is_active, created_at, updated_at
			FROM billing_resolutions WHERE id = $1`
		res, err = scanResolution(r.pool.QueryRow(ctx, qWithoutIsActive, id))
	}
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
	const qWithIsActive = `
		SELECT id, company_id, resolution_number, prefix, range_from, range_to,
		       date_from, date_to, environment, 0::bigint AS used_numbers, is_active, created_at, updated_at
		FROM billing_resolutions
		WHERE company_id = $1
		  AND prefix     = $2
		  AND is_active  = true
		  AND date_to   >= CURRENT_DATE
		ORDER BY date_from DESC
		LIMIT 1`
	res, err := scanResolution(r.pool.QueryRow(ctx, qWithIsActive, companyID, prefix))
	if isUndefinedColumnError(err, "is_active") {
		const qWithoutIsActive = `
			SELECT id, company_id, resolution_number, prefix, range_from, range_to,
			       date_from, date_to, environment, 0::bigint AS used_numbers, true AS is_active, created_at, updated_at
			FROM billing_resolutions
			WHERE company_id = $1
			  AND prefix     = $2
			  AND date_to   >= CURRENT_DATE
			ORDER BY date_from DESC
			LIMIT 1`
		res, err = scanResolution(r.pool.QueryRow(ctx, qWithoutIsActive, companyID, prefix))
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // no hay resolución activa: factura sin DianExtensions
		}
		return nil, fmt.Errorf("get active billing_resolution: %w", err)
	}
	return res, nil
}

func (r *BillingResolutionRepo) ListByCompany(ctx context.Context, companyID string) ([]*entity.BillingResolution, error) {
	const qWithIsActive = `
		SELECT br.id, br.company_id, br.resolution_number, br.prefix, br.range_from, br.range_to,
		       br.date_from, br.date_to, br.environment,
		       COALESCE((
			   SELECT COUNT(1)::bigint
			   FROM invoices i
			   WHERE i.company_id = br.company_id
				 AND i.prefix = br.prefix
				 AND i.number ~ '^[0-9]+$'
				 AND i.number::bigint BETWEEN br.range_from AND br.range_to
		   ), 0) AS used_numbers,
		       br.is_active, br.created_at, br.updated_at
		FROM billing_resolutions br
		WHERE br.company_id = $1
		ORDER BY br.date_from DESC`
	rows, err := r.pool.Query(ctx, qWithIsActive, companyID)
	if isUndefinedColumnError(err, "is_active") {
		const qWithoutIsActive = `
			SELECT br.id, br.company_id, br.resolution_number, br.prefix, br.range_from, br.range_to,
			       br.date_from, br.date_to, br.environment,
			       COALESCE((
				   SELECT COUNT(1)::bigint
				   FROM invoices i
				   WHERE i.company_id = br.company_id
					 AND i.prefix = br.prefix
					 AND i.number ~ '^[0-9]+$'
					 AND i.number::bigint BETWEEN br.range_from AND br.range_to
			   ), 0) AS used_numbers,
			       true AS is_active, br.created_at, br.updated_at
			FROM billing_resolutions br
			WHERE br.company_id = $1
			ORDER BY br.date_from DESC`
		rows, err = r.pool.Query(ctx, qWithoutIsActive, companyID)
	}
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
	const qWithIsActive = `
		UPDATE billing_resolutions
		SET resolution_number = $2, prefix = $3, range_from = $4, range_to = $5,
		    date_from = $6, date_to = $7, environment = $8, is_active = $9, updated_at = now()
		WHERE id = $1`
	_, err := r.pool.Exec(ctx, qWithIsActive,
		res.ID, res.ResolutionNumber, res.Prefix,
		res.RangeFrom, res.RangeTo, res.DateFrom, res.DateTo, res.Environment, res.IsActive,
	)
	if isUndefinedColumnError(err, "is_active") {
		const qWithoutIsActive = `
			UPDATE billing_resolutions
			SET resolution_number = $2, prefix = $3, range_from = $4, range_to = $5,
			    date_from = $6, date_to = $7, environment = $8, updated_at = now()
			WHERE id = $1`
		_, err = r.pool.Exec(ctx, qWithoutIsActive,
			res.ID, res.ResolutionNumber, res.Prefix,
			res.RangeFrom, res.RangeTo, res.DateFrom, res.DateTo, res.Environment,
		)
	}
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
		&res.Environment, &res.UsedNumbers,
		&res.IsActive, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func isUndefinedColumnError(err error, column string) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "42703" && strings.Contains(strings.ToLower(pgErr.Message), strings.ToLower(column))
}
