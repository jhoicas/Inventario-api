-- 038 - Hybrid email account config (OAuth + custom IMAP/SMTP)

ALTER TABLE email_accounts
    ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS provider VARCHAR(20) NOT NULL DEFAULT 'custom',
    ADD COLUMN IF NOT EXISTS access_token TEXT,
    ADD COLUMN IF NOT EXISTS refresh_token TEXT,
    ADD COLUMN IF NOT EXISTS imap_host VARCHAR(255),
    ADD COLUMN IF NOT EXISTS smtp_host VARCHAR(255),
    ADD COLUMN IF NOT EXISTS smtp_port INT,
    ADD COLUMN IF NOT EXISTS app_password VARCHAR(1024);

UPDATE email_accounts
SET imap_host = COALESCE(imap_host, imap_server),
    app_password = COALESCE(app_password, password)
WHERE imap_host IS NULL OR app_password IS NULL;

CREATE INDEX IF NOT EXISTS idx_email_accounts_provider
    ON email_accounts(provider);

CREATE INDEX IF NOT EXISTS idx_email_accounts_company_user_provider
    ON email_accounts(company_id, user_id, provider);
