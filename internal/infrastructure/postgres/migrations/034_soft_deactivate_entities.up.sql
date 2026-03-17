-- 034_soft_deactivate_entities.up.sql
-- Agrega columna is_active para desactivar sin borrar.

ALTER TABLE crm_categories
ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE customers
ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE suppliers
ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_customers_company_is_active ON customers(company_id, is_active);
CREATE INDEX IF NOT EXISTS idx_suppliers_company_is_active ON suppliers(company_id, is_active);
CREATE INDEX IF NOT EXISTS idx_crm_categories_company_is_active ON crm_categories(company_id, is_active);

