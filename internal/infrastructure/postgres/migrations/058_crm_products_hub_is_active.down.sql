DROP INDEX IF EXISTS idx_crm_products_hub_company_active;
ALTER TABLE crm_products_hub DROP COLUMN IF EXISTS is_active;
