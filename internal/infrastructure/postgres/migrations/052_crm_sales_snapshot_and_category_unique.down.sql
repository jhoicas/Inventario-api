-- 052_crm_sales_snapshot_and_category_unique.down.sql

DROP INDEX IF EXISTS uq_crm_categories_company_name;

ALTER TABLE crm_sales_hub
    DROP COLUMN IF EXISTS items_snapshot;
