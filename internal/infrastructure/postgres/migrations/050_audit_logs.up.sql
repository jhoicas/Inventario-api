-- 050_audit_logs.up.sql

CREATE TABLE IF NOT EXISTS audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id     UUID,
    action      VARCHAR(20) NOT NULL,
    entity_name VARCHAR(100) NOT NULL,
    entity_id   VARCHAR(100) NOT NULL,
    changes     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_company_created_at
    ON audit_logs(company_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id
    ON audit_logs(user_id);

CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_name
    ON audit_logs(entity_name);
