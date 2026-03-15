ALTER TABLE dian_settings DROP CONSTRAINT IF EXISTS dian_settings_pkey;

WITH ranked AS (
    SELECT
        ctid,
        ROW_NUMBER() OVER (
            PARTITION BY company_id
            ORDER BY updated_at DESC, created_at DESC, environment DESC
        ) AS rn
    FROM dian_settings
)
DELETE FROM dian_settings d
USING ranked r
WHERE d.ctid = r.ctid
  AND r.rn > 1;

ALTER TABLE dian_settings
    ADD CONSTRAINT dian_settings_pkey PRIMARY KEY (company_id);