package billing

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jhoicas/Inventario-api/internal/models"
)

type Resolution = models.Resolution

type BillingResolutionRepository struct {
	pool *pgxpool.Pool
}

func NewBillingResolutionRepository(pool *pgxpool.Pool) *BillingResolutionRepository {
	return &BillingResolutionRepository{pool: pool}
}

func (r *BillingResolutionRepository) GetByCompanyID(companyID string) ([]Resolution, error) {
	const q = `
		SELECT br.id,
		       br.company_id,
		       br.prefix,
		       br.resolution_number,
		       br.range_from,
		       br.range_to,
		       br.date_from,
		       br.date_to,
		       br.environment,
		       COALESCE((
		           SELECT COUNT(1)::bigint
		           FROM invoices i
		           WHERE i.company_id = br.company_id
		             AND i.prefix = br.prefix
		             AND COALESCE(i.document_type, 'INVOICE') = 'INVOICE'
		       ), 0) AS used_numbers
		FROM billing_resolutions br
		WHERE br.company_id = $1
		ORDER BY br.date_from DESC`

	rows, err := r.pool.Query(context.Background(), q, companyID)
	if err != nil {
		return nil, fmt.Errorf("list billing resolutions by company: %w", err)
	}
	defer rows.Close()

	resolutions := make([]Resolution, 0)
	for rows.Next() {
		var item Resolution
		if err := rows.Scan(
			&item.ID,
			&item.CompanyID,
			&item.Prefix,
			&item.ResolutionNumber,
			&item.FromNumber,
			&item.ToNumber,
			&item.ValidFrom,
			&item.ValidUntil,
			&item.Environment,
			&item.UsedNumbers,
		); err != nil {
			return nil, fmt.Errorf("scan billing resolution: %w", err)
		}
		resolutions = append(resolutions, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate billing resolutions: %w", err)
	}

	return resolutions, nil
}

func (r *BillingResolutionRepository) Create(resolution *Resolution) error {
	const q = `
		INSERT INTO billing_resolutions (
			id,
			company_id,
			prefix,
			resolution_number,
			range_from,
			range_to,
			date_from,
			date_to,
			environment,
			is_active,
			created_at,
			updated_at
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			now(),
			now()
		)`

	_, err := r.pool.Exec(
		context.Background(),
		q,
		resolution.ID,
		resolution.CompanyID,
		resolution.Prefix,
		resolution.ResolutionNumber,
		resolution.FromNumber,
		resolution.ToNumber,
		resolution.ValidFrom,
		resolution.ValidUntil,
		resolution.Environment,
		true,
	)
	if err != nil {
		return fmt.Errorf("insert billing resolution: %w", err)
	}

	return nil
}
