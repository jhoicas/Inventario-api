-- 033_crm_campaign_templates.up.sql

CREATE TABLE IF NOT EXISTS crm_campaign_templates (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name       VARCHAR(200) NOT NULL,
    subject    VARCHAR(255) NOT NULL,
    body       TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_crm_campaign_templates_company_id
    ON crm_campaign_templates(company_id);

