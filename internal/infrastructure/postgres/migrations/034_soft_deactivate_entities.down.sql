-- 034_soft_deactivate_entities.down.sql

ALTER TABLE crm_categories DROP COLUMN IF EXISTS is_active;
ALTER TABLE customers DROP COLUMN IF EXISTS is_active;
ALTER TABLE suppliers DROP COLUMN IF EXISTS is_active;

DROP INDEX IF EXISTS idx_customers_company_is_active;
DROP INDEX IF EXISTS idx_suppliers_company_is_active;
DROP INDEX IF EXISTS idx_crm_categories_company_is_active;

