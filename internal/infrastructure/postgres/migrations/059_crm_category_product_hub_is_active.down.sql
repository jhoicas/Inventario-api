DROP INDEX IF EXISTS idx_crm_category_product_hub_company_active;

ALTER TABLE crm_category_product_hub
DROP COLUMN IF EXISTS is_active;
