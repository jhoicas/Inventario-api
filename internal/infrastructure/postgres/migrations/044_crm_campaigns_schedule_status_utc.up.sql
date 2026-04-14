-- 044_crm_campaigns_schedule_status_utc.up.sql

-- Ensure campaign scheduling fields exist and default to deferred processing status.
ALTER TABLE crm_campaigns
  ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS status VARCHAR(20);

-- Replace legacy status CHECK so 'pending' is accepted while preserving old values.
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
  ALTER COLUMN status SET DEFAULT 'pending';

UPDATE crm_campaigns
SET status = 'pending'
WHERE status IS NULL OR btrim(status) = '';

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid = 'crm_campaigns'::regclass
      AND conname = 'crm_campaigns_status_check'
  ) THEN
    ALTER TABLE crm_campaigns
      ADD CONSTRAINT crm_campaigns_status_check
      CHECK (status IN ('pending', 'scheduled', 'sending', 'completed', 'failed', 'cancelled', 'BORRADOR', 'PROGRAMADA', 'ENVIANDO', 'COMPLETADA'));
  END IF;
END $$;

ALTER TABLE crm_campaigns
  ALTER COLUMN status SET NOT NULL;
