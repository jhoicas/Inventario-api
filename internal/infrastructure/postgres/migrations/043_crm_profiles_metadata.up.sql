-- 043_crm_profiles_metadata.up.sql
-- Agrega metadata JSONB para perfiles CRM en entornos ya migrados.

ALTER TABLE crm_customer_profiles
ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;
