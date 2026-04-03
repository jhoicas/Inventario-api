-- 042_company_screens.up.sql
-- Pantallas habilitadas por empresa para control multitenant de RBAC.

CREATE TABLE IF NOT EXISTS company_screens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    screen_id    UUID NOT NULL REFERENCES screens(id) ON DELETE CASCADE,
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id, screen_id)
);

CREATE INDEX IF NOT EXISTS idx_company_screens_company_id ON company_screens (company_id);
CREATE INDEX IF NOT EXISTS idx_company_screens_screen_id ON company_screens (screen_id);
CREATE INDEX IF NOT EXISTS idx_company_screens_active ON company_screens (company_id, is_active);
