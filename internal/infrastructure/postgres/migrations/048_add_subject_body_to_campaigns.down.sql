-- 048_add_subject_body_to_campaigns.down.sql
ALTER TABLE crm_campaigns DROP COLUMN IF EXISTS subject;
ALTER TABLE crm_campaigns DROP COLUMN IF EXISTS body;
