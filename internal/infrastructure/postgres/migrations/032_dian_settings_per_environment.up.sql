ALTER TABLE dian_settings DROP CONSTRAINT IF EXISTS dian_settings_pkey;

ALTER TABLE dian_settings
    ADD CONSTRAINT dian_settings_pkey PRIMARY KEY (company_id, environment);

CREATE INDEX IF NOT EXISTS idx_dian_settings_company_id ON dian_settings(company_id);