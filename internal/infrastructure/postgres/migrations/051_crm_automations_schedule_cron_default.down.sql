-- 051_crm_automations_schedule_cron_default.down.sql

ALTER TABLE crm_automations
    ALTER COLUMN schedule_cron DROP NOT NULL;

ALTER TABLE crm_automations
    ALTER COLUMN schedule_cron DROP DEFAULT;
