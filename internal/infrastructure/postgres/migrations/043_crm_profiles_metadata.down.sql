-- 043_crm_profiles_metadata.down.sql
-- Revierte metadata JSONB en perfiles CRM.

ALTER TABLE crm_customer_profiles
DROP COLUMN IF EXISTS metadata;
