CREATE TABLE IF NOT EXISTS notification_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    customer_id UUID NULL REFERENCES customers(id) ON DELETE SET NULL,
    type VARCHAR(32) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    subject TEXT NULL,
    body TEXT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status VARCHAR(32) NOT NULL,
    error_message TEXT NULL
);

CREATE INDEX IF NOT EXISTS idx_notification_logs_company_sent_at
    ON notification_logs (company_id, sent_at DESC);

CREATE INDEX IF NOT EXISTS idx_notification_logs_type
    ON notification_logs (type);
