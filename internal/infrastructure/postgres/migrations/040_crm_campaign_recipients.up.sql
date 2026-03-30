-- 040_crm_campaign_recipients.up.sql

CREATE TABLE IF NOT EXISTS crm_campaign_recipients (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id    UUID         NOT NULL REFERENCES crm_campaigns(id) ON DELETE CASCADE,
    customer_id    UUID         NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    company_id     UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    email          VARCHAR(320) NOT NULL,
    subject        VARCHAR(500) NOT NULL,
    body           TEXT         NOT NULL,
    status         VARCHAR(20)  NOT NULL DEFAULT 'QUEUED' CHECK (status IN ('QUEUED', 'SENT', 'FAILED')),
    error_message  TEXT,
    queued_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    sent_at        TIMESTAMPTZ,
    processed_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_crm_campaign_recipients_campaign_id
    ON crm_campaign_recipients(campaign_id);

CREATE INDEX IF NOT EXISTS idx_crm_campaign_recipients_company_status
    ON crm_campaign_recipients(company_id, status);

CREATE INDEX IF NOT EXISTS idx_crm_campaign_recipients_customer_id
    ON crm_campaign_recipients(customer_id);

CREATE INDEX IF NOT EXISTS idx_crm_campaign_recipients_queued_at
    ON crm_campaign_recipients(queued_at DESC);
