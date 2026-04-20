-- Flag de catálogo en hub CRM (desactivar sin borrar fila).
ALTER TABLE crm_products_hub
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_crm_products_hub_company_active
    ON crm_products_hub (company_id, is_active);
