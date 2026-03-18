-- 036 - Soft delete para crm_benefits (is_active)

ALTER TABLE crm_benefits
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_crm_benefits_company_active
    ON crm_benefits(company_id, is_active);

