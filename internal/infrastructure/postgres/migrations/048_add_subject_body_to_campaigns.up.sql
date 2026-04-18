-- 048_add_subject_body_to_campaigns.up.sql
ALTER TABLE crm_campaigns ADD COLUMN IF NOT EXISTS subject TEXT;
ALTER TABLE crm_campaigns ADD COLUMN IF NOT EXISTS body TEXT;
