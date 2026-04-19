-- 051_crm_automations_schedule_cron_default.up.sql

UPDATE crm_automations
SET schedule_cron = '0 0 * * *'
WHERE schedule_cron IS NULL OR BTRIM(schedule_cron) = '';

ALTER TABLE crm_automations
    ALTER COLUMN schedule_cron SET DEFAULT '0 0 * * *';

ALTER TABLE crm_automations
    ALTER COLUMN schedule_cron SET NOT NULL;
