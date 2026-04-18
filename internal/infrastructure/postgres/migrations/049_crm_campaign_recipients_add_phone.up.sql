-- 049_crm_campaign_recipients_add_phone.up.sql

ALTER TABLE crm_campaign_recipients
ADD COLUMN IF NOT EXISTS phone VARCHAR(50) NULL;

-- Create index for phone column to support filtering by phone for SMS/WhatsApp deliveries
CREATE INDEX IF NOT EXISTS idx_crm_campaign_recipients_phone
    ON crm_campaign_recipients(phone)
    WHERE phone IS NOT NULL AND phone != '';
