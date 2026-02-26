-- Tabla de resoluciones de facturación autorizadas por la DIAN.
-- Es obligatoria para incluir sts:DianExtensions en el XML UBL 2.1.
-- Una empresa puede tener múltiples resoluciones (historial), pero solo
-- una activa por prefijo a la vez (índice único parcial).

CREATE TABLE IF NOT EXISTS billing_resolutions (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id        UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    resolution_number VARCHAR(50) NOT NULL,                    -- Ej: "18764000000001"
    prefix            VARCHAR(10) NOT NULL,                    -- Ej: "SETP", "FE", "FV"
    range_from        BIGINT      NOT NULL DEFAULT 1,          -- Número inicial autorizado
    range_to          BIGINT      NOT NULL DEFAULT 99999999,   -- Número final autorizado
    date_from         DATE        NOT NULL,                    -- Inicio de vigencia
    date_to           DATE        NOT NULL,                    -- Vencimiento
    is_active         BOOLEAN     NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Solo puede haber una resolución activa por (empresa, prefijo) a la vez.
CREATE UNIQUE INDEX IF NOT EXISTS uq_billing_resolutions_active_prefix
    ON billing_resolutions (company_id, prefix)
    WHERE is_active = true;

-- Índice para consultas frecuentes (resolución activa de una empresa).
CREATE INDEX IF NOT EXISTS idx_billing_resolutions_company
    ON billing_resolutions (company_id, is_active, date_to DESC);
