-- 047_crm_campaigns_multichannel.down.sql

ALTER TABLE crm_campaigns DROP COLUMN IF EXISTS channel;
