-- 044_crm_campaigns_schedule_status_utc.down.sql

-- Revert to legacy status domain used by original CRM campaign migration.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid = 'crm_campaigns'::regclass
      AND conname = 'crm_campaigns_status_check'
  ) THEN
    ALTER TABLE crm_campaigns DROP CONSTRAINT crm_campaigns_status_check;
  END IF;
END $$;

ALTER TABLE crm_campaigns
  ADD CONSTRAINT crm_campaigns_status_check
  CHECK (status IN ('BORRADOR', 'PROGRAMADA', 'ENVIANDO', 'COMPLETADA'));

ALTER TABLE crm_campaigns
  ALTER COLUMN status DROP DEFAULT;
