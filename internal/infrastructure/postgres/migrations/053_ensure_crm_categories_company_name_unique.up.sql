-- 053_ensure_crm_categories_company_name_unique.up.sql
-- Garantiza índice único (company_id, name) para INSERT ... ON CONFLICT y para evitar duplicados
-- en importaciones CRM. Idempotente: coincide con 052 si esa migración ya se aplicó.

CREATE UNIQUE INDEX IF NOT EXISTS uq_crm_categories_company_name
    ON crm_categories (company_id, name);
