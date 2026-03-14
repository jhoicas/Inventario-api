-- 027_sla_config.down.sql

DROP TABLE IF EXISTS sla_config;

ALTER TABLE crm_tickets DROP COLUMN IF EXISTS escalation_reason;
