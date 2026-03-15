CREATE TABLE IF NOT EXISTS dian_settings (
    company_id                    UUID PRIMARY KEY REFERENCES companies(id) ON DELETE CASCADE,
    environment                   VARCHAR(10)   NOT NULL CHECK (environment IN ('test', 'prod')),
    certificate_path              TEXT          NOT NULL,
    certificate_file_name         VARCHAR(255)  NOT NULL,
    certificate_file_size         BIGINT        NOT NULL CHECK (certificate_file_size > 0),
    certificate_password_encrypted TEXT         NOT NULL,
    created_at                    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at                    TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_dian_settings_environment ON dian_settings(environment);
