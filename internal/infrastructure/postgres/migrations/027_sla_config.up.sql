-- 027_sla_config.up.sql

ALTER TABLE crm_tickets ADD COLUMN IF NOT EXISTS escalation_reason TEXT;

CREATE TABLE IF NOT EXISTS sla_config (
    company_id   UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    ticket_type  VARCHAR(50)  NOT NULL DEFAULT '',
    max_hours    INT          NOT NULL CHECK (max_hours > 0),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (company_id, ticket_type)
);

CREATE INDEX IF NOT EXISTS idx_sla_config_company_id ON sla_config(company_id);
