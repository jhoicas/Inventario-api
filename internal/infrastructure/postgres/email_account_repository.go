package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

var _ repository.HybridEmailAccountRepository = (*HybridEmailAccountRepo)(nil)

// HybridEmailAccountRepo persistencia PostgreSQL para configuración híbrida de correo.
type HybridEmailAccountRepo struct {
	pool *pgxpool.Pool
}

func NewHybridEmailAccountRepository(pool *pgxpool.Pool) *HybridEmailAccountRepo {
	return &HybridEmailAccountRepo{pool: pool}
}

func (r *HybridEmailAccountRepo) Save(ctx context.Context, account *entity.EmailAccountConfig) error {
	if account == nil {
		return fmt.Errorf("email account config nil")
	}
	if account.ID == "" {
		account.ID = uuid.New().String()
	}
	if account.CreatedAt.IsZero() {
		account.CreatedAt = time.Now().UTC()
	}
	if account.UpdatedAt.IsZero() {
		account.UpdatedAt = time.Now().UTC()
	}

	provider := strings.TrimSpace(strings.ToLower(account.Provider))
	if provider == "" {
		provider = "custom"
	}

	legacyIMAPServer := strings.TrimSpace(account.ImapHost)
	legacyPassword := strings.TrimSpace(account.AppPassword)

	err := r.pool.QueryRow(ctx, `
		INSERT INTO email_accounts (
			id, user_id, company_id, provider, email_address,
			access_token, refresh_token,
			imap_host, imap_port, smtp_host, smtp_port, app_password,
			is_active, created_at, updated_at,
			imap_server, password
		)
		VALUES (
			$1,
			NULLIF($2, '')::uuid,
			$3,
			$4,
			$5,
			NULLIF($6, ''),
			NULLIF($7, ''),
			NULLIF($8, ''),
			$9,
			NULLIF($10, ''),
			$11,
			NULLIF($12, ''),
			$13,
			$14,
			$15,
			$16,
			$17
		)
		ON CONFLICT (company_id, email_address)
		DO UPDATE SET
			user_id = NULLIF(EXCLUDED.user_id::text, '')::uuid,
			provider = EXCLUDED.provider,
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			imap_host = EXCLUDED.imap_host,
			imap_port = EXCLUDED.imap_port,
			smtp_host = EXCLUDED.smtp_host,
			smtp_port = EXCLUDED.smtp_port,
			app_password = EXCLUDED.app_password,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at,
			imap_server = EXCLUDED.imap_server,
			password = EXCLUDED.password
		RETURNING id, created_at, updated_at`,
		account.ID,
		strings.TrimSpace(account.UserID),
		account.CompanyID,
		provider,
		strings.TrimSpace(strings.ToLower(account.EmailAddress)),
		strings.TrimSpace(account.AccessToken),
		strings.TrimSpace(account.RefreshToken),
		strings.TrimSpace(account.ImapHost),
		account.ImapPort,
		strings.TrimSpace(account.SmtpHost),
		account.SmtpPort,
		strings.TrimSpace(account.AppPassword),
		account.IsActive,
		account.CreatedAt,
		account.UpdatedAt,
		legacyIMAPServer,
		legacyPassword,
	).Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save hybrid email account: %w", err)
	}
	account.Provider = provider
	account.EmailAddress = strings.TrimSpace(strings.ToLower(account.EmailAddress))
	return nil
}

func (r *HybridEmailAccountRepo) GetByID(ctx context.Context, companyID, id string) (*entity.EmailAccountConfig, error) {
	var out entity.EmailAccountConfig
	var userID *string
	err := r.pool.QueryRow(ctx, `
		SELECT
			id,
			user_id::text,
			company_id,
			COALESCE(provider, 'custom'),
			email_address,
			COALESCE(access_token, ''),
			COALESCE(refresh_token, ''),
			COALESCE(imap_host, imap_server, ''),
			COALESCE(imap_port, 0),
			COALESCE(smtp_host, ''),
			COALESCE(smtp_port, 0),
			COALESCE(app_password, password, ''),
			is_active,
			created_at,
			updated_at
		FROM email_accounts
		WHERE company_id = $1 AND id = $2`, companyID, id,
	).Scan(
		&out.ID,
		&userID,
		&out.CompanyID,
		&out.Provider,
		&out.EmailAddress,
		&out.AccessToken,
		&out.RefreshToken,
		&out.ImapHost,
		&out.ImapPort,
		&out.SmtpHost,
		&out.SmtpPort,
		&out.AppPassword,
		&out.IsActive,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get hybrid email account by id: %w", err)
	}
	if userID != nil {
		out.UserID = *userID
	}
	return &out, nil
}

func (r *HybridEmailAccountRepo) GetByCompanyAndEmail(ctx context.Context, companyID, emailAddress string) (*entity.EmailAccountConfig, error) {
	var out entity.EmailAccountConfig
	var userID *string
	err := r.pool.QueryRow(ctx, `
		SELECT
			id,
			user_id::text,
			company_id,
			COALESCE(provider, 'custom'),
			email_address,
			COALESCE(access_token, ''),
			COALESCE(refresh_token, ''),
			COALESCE(imap_host, imap_server, ''),
			COALESCE(imap_port, 0),
			COALESCE(smtp_host, ''),
			COALESCE(smtp_port, 0),
			COALESCE(app_password, password, ''),
			is_active,
			created_at,
			updated_at
		FROM email_accounts
		WHERE company_id = $1 AND email_address = $2`,
		companyID,
		strings.TrimSpace(strings.ToLower(emailAddress)),
	).Scan(
		&out.ID,
		&userID,
		&out.CompanyID,
		&out.Provider,
		&out.EmailAddress,
		&out.AccessToken,
		&out.RefreshToken,
		&out.ImapHost,
		&out.ImapPort,
		&out.SmtpHost,
		&out.SmtpPort,
		&out.AppPassword,
		&out.IsActive,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get hybrid email account by email: %w", err)
	}
	if userID != nil {
		out.UserID = *userID
	}
	return &out, nil
}

func (r *HybridEmailAccountRepo) GetByCompanyAndProvider(ctx context.Context, companyID, provider string) (*entity.EmailAccountConfig, error) {
	var out entity.EmailAccountConfig
	var userID *string
	err := r.pool.QueryRow(ctx, `
		SELECT
			id,
			user_id::text,
			company_id,
			COALESCE(provider, 'custom'),
			email_address,
			COALESCE(access_token, ''),
			COALESCE(refresh_token, ''),
			COALESCE(imap_host, imap_server, ''),
			COALESCE(imap_port, 0),
			COALESCE(smtp_host, ''),
			COALESCE(smtp_port, 0),
			COALESCE(app_password, password, ''),
			is_active,
			created_at,
			updated_at
		FROM email_accounts
		WHERE company_id = $1 AND LOWER(COALESCE(provider, 'custom')) = LOWER($2)
		ORDER BY is_active DESC, updated_at DESC
		LIMIT 1`,
		companyID,
		strings.TrimSpace(provider),
	).Scan(
		&out.ID,
		&userID,
		&out.CompanyID,
		&out.Provider,
		&out.EmailAddress,
		&out.AccessToken,
		&out.RefreshToken,
		&out.ImapHost,
		&out.ImapPort,
		&out.SmtpHost,
		&out.SmtpPort,
		&out.AppPassword,
		&out.IsActive,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get hybrid email account by provider: %w", err)
	}
	if userID != nil {
		out.UserID = *userID
	}
	return &out, nil
}
