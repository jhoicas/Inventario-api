-- 038 rollback - Hybrid email account config

DROP INDEX IF EXISTS idx_email_accounts_company_user_provider;
DROP INDEX IF EXISTS idx_email_accounts_provider;

ALTER TABLE email_accounts
    DROP COLUMN IF EXISTS app_password,
    DROP COLUMN IF EXISTS smtp_port,
    DROP COLUMN IF EXISTS smtp_host,
    DROP COLUMN IF EXISTS imap_host,
    DROP COLUMN IF EXISTS refresh_token,
    DROP COLUMN IF EXISTS access_token,
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS user_id;
