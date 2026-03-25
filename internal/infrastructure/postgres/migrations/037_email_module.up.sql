-- 037 - Email inbox module (IMAP read + CRM integration)

CREATE TABLE IF NOT EXISTS email_accounts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    email_address VARCHAR(255) NOT NULL,
    imap_server   VARCHAR(255) NOT NULL,
    imap_port     INT          NOT NULL,
    password      VARCHAR(1024) NOT NULL,
    is_active     BOOLEAN      NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_email_accounts_company_email
    ON email_accounts(company_id, email_address);
CREATE INDEX IF NOT EXISTS idx_email_accounts_company_active
    ON email_accounts(company_id, is_active);

CREATE TABLE IF NOT EXISTS emails (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id   UUID         NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    message_id   VARCHAR(500) NOT NULL,
    customer_id  UUID         REFERENCES customers(id) ON DELETE SET NULL,
    from_address VARCHAR(500) NOT NULL,
    to_address   VARCHAR(1000) NOT NULL,
    subject      VARCHAR(500) NOT NULL,
    body_html    TEXT,
    body_text    TEXT,
    received_at  TIMESTAMPTZ  NOT NULL,
    is_read      BOOLEAN      NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_emails_account_message_id
    ON emails(account_id, message_id);
CREATE INDEX IF NOT EXISTS idx_emails_account_received_at
    ON emails(account_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_emails_customer_id
    ON emails(customer_id);
CREATE INDEX IF NOT EXISTS idx_emails_is_read
    ON emails(is_read);

CREATE TABLE IF NOT EXISTS email_attachments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email_id   UUID         NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    file_name  VARCHAR(500) NOT NULL,
    file_url   TEXT         NOT NULL DEFAULT '',
    mime_type  VARCHAR(255) NOT NULL,
    size       INT          NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_email_attachments_email_id
    ON email_attachments(email_id);
