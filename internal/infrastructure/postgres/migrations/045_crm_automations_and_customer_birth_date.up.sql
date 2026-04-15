-- 045_crm_automations_and_customer_birth_date.up.sql

ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS birth_date DATE;

CREATE INDEX IF NOT EXISTS idx_customers_company_birth_date
    ON customers(company_id, birth_date);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_type WHERE typname = 'crm_automation_type'
    ) THEN
        CREATE TYPE crm_automation_type AS ENUM ('BIRTHDAY', 'REPURCHASE');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS crm_automations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name          VARCHAR(200) NOT NULL,
    type          crm_automation_type NOT NULL,
    template_id   UUID REFERENCES crm_campaign_templates(id) ON DELETE SET NULL,
    schedule_cron VARCHAR(100),
    config        JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    last_run_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_crm_automations_company_id
    ON crm_automations(company_id);

CREATE INDEX IF NOT EXISTS idx_crm_automations_active
    ON crm_automations(is_active);

CREATE INDEX IF NOT EXISTS idx_crm_automations_type
    ON crm_automations(type);
