ALTER TABLE crm_category_product_hub
ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_crm_category_product_hub_company_active
ON crm_category_product_hub (company_id, is_active);
