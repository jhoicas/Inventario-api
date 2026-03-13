CREATE TABLE IF NOT EXISTS suppliers (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id        UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name              VARCHAR(200)  NOT NULL,
    nit               VARCHAR(50)   NOT NULL,
    email             VARCHAR(255),
    phone             VARCHAR(50),
    payment_term_days INTEGER       NOT NULL DEFAULT 0 CHECK (payment_term_days >= 0),
    lead_time_days    INTEGER       NOT NULL DEFAULT 0 CHECK (lead_time_days >= 0),
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ   NOT NULL DEFAULT now(),
    UNIQUE (company_id, nit)
);

CREATE INDEX IF NOT EXISTS idx_suppliers_company_id ON suppliers(company_id);
CREATE INDEX IF NOT EXISTS idx_suppliers_company_nit ON suppliers(company_id, nit);
