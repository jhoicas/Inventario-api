-- 029_crm_campaigns.up.sql

CREATE TABLE IF NOT EXISTS crm_campaigns (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name         VARCHAR(200) NOT NULL,
    description  TEXT,
    status       VARCHAR(20)  NOT NULL CHECK (status IN ('BORRADOR', 'PROGRAMADA', 'ENVIANDO', 'COMPLETADA')),
    scheduled_at TIMESTAMPTZ,
    created_by   UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_crm_campaigns_company_id ON crm_campaigns(company_id);
CREATE INDEX IF NOT EXISTS idx_crm_campaigns_status ON crm_campaigns(status);
CREATE INDEX IF NOT EXISTS idx_crm_campaigns_scheduled_at ON crm_campaigns(scheduled_at);

CREATE TABLE IF NOT EXISTS crm_campaign_metrics (
    campaign_id UUID PRIMARY KEY REFERENCES crm_campaigns(id) ON DELETE CASCADE,
    sent        INT            NOT NULL DEFAULT 0,
    opened      INT            NOT NULL DEFAULT 0,
    clicked     INT            NOT NULL DEFAULT 0,
    converted   INT            NOT NULL DEFAULT 0,
    revenue     DECIMAL(15,2)  NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ    NOT NULL DEFAULT now()
);
