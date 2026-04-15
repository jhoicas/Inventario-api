-- Rollback CRM AI Analytics Hub Tables and Semantic Layer

DROP FUNCTION IF EXISTS crm_ai_analytics_with_company(UUID) CASCADE;
DROP VIEW IF EXISTS v_crm_ai_analytics CASCADE;
DROP TABLE IF EXISTS crm_sale_items_hub CASCADE;
DROP TABLE IF EXISTS crm_sales_hub CASCADE;
DROP TABLE IF EXISTS crm_products_hub CASCADE;
