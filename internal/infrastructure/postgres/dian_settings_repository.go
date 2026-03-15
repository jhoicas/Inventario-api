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

var _ repository.DIANSettingsRepository = (*DIANSettingsRepo)(nil)

// DIANSettingsRepo implementa DIANSettingsRepository sobre PostgreSQL.
type DIANSettingsRepo struct {
	pool *pgxpool.Pool
}

func NewDIANSettingsRepository(pool *pgxpool.Pool) *DIANSettingsRepo {
	return &DIANSettingsRepo{pool: pool}
}

func (r *DIANSettingsRepo) Upsert(ctx context.Context, settings *entity.DIANSettings) error {
	const q = `
		INSERT INTO dian_settings (
			company_id,
			environment,
			certificate_path,
			certificate_file_name,
			certificate_file_size,
			certificate_password_encrypted,
			created_at,
			updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (company_id) DO UPDATE SET
			environment = EXCLUDED.environment,
			certificate_path = EXCLUDED.certificate_path,
			certificate_file_name = EXCLUDED.certificate_file_name,
			certificate_file_size = EXCLUDED.certificate_file_size,
			certificate_password_encrypted = EXCLUDED.certificate_password_encrypted,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.pool.Exec(
		ctx,
		q,
		settings.CompanyID,
		settings.Environment,
		settings.CertificatePath,
		settings.CertificateFileName,
		settings.CertificateFileSize,
		settings.CertificatePasswordEncrypted,
		settings.CreatedAt,
		settings.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert dian_settings: %w", err)
	}
	return nil
}

func (r *DIANSettingsRepo) GetByCompanyID(ctx context.Context, companyID string) (*entity.DIANSettings, error) {
	const q = `
		SELECT
			company_id,
			environment,
			certificate_path,
			certificate_file_name,
			certificate_file_size,
			certificate_password_encrypted,
			created_at,
			updated_at
		FROM dian_settings
		WHERE company_id = $1
	`

	var settings entity.DIANSettings
	err := r.pool.QueryRow(ctx, q, companyID).Scan(
		&settings.CompanyID,
		&settings.Environment,
		&settings.CertificatePath,
		&settings.CertificateFileName,
		&settings.CertificateFileSize,
		&settings.CertificatePasswordEncrypted,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get dian_settings by company_id: %w", err)
	}

	return &settings, nil
}
