-- 049_crm_campaign_recipients_add_phone.down.sql

ALTER TABLE crm_campaign_recipients
DROP COLUMN IF EXISTS phone;
