-- 047_crm_campaigns_multichannel.up.sql

ALTER TABLE crm_campaigns ADD COLUMN IF NOT EXISTS channel VARCHAR(20) NOT NULL DEFAULT 'EMAIL' CHECK (channel IN ('EMAIL', 'SMS', 'WHATSAPP'));
