-- 045_crm_automations_and_customer_birth_date.down.sql

DROP TABLE IF EXISTS crm_automations;
DROP TYPE IF EXISTS crm_automation_type;
DROP INDEX IF EXISTS idx_customers_company_birth_date;
ALTER TABLE customers DROP COLUMN IF EXISTS birth_date;
