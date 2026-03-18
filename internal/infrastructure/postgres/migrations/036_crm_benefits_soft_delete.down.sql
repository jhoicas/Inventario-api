-- 036 - Rollback soft delete crm_benefits

DROP INDEX IF EXISTS idx_crm_benefits_company_active;

ALTER TABLE crm_benefits
    DROP COLUMN IF EXISTS is_active;

