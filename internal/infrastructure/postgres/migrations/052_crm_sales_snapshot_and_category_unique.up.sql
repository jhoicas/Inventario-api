-- 052_crm_sales_snapshot_and_category_unique.up.sql

ALTER TABLE crm_sales_hub
    ADD COLUMN IF NOT EXISTS items_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE UNIQUE INDEX IF NOT EXISTS uq_crm_categories_company_name
    ON crm_categories(company_id, name);
