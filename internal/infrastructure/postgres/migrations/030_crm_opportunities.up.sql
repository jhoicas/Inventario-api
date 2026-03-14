-- 030_crm_opportunities.up.sql

CREATE TABLE IF NOT EXISTS crm_opportunities (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id          UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    customer_id         UUID          REFERENCES customers(id) ON DELETE SET NULL,
    title               VARCHAR(300)  NOT NULL,
    amount              DECIMAL(15,2) NOT NULL DEFAULT 0,
    probability         INT           NOT NULL DEFAULT 0 CHECK (probability >= 0 AND probability <= 100),
    stage               VARCHAR(20)   NOT NULL DEFAULT 'prospecto'
                                   CHECK (stage IN ('prospecto', 'calificado', 'propuesta', 'negociacion', 'ganado', 'perdido')),
    expected_close_date TIMESTAMPTZ,
    created_by          UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_crm_opportunities_company_id ON crm_opportunities(company_id);
CREATE INDEX IF NOT EXISTS idx_crm_opportunities_stage ON crm_opportunities(stage);
CREATE INDEX IF NOT EXISTS idx_crm_opportunities_customer_id ON crm_opportunities(customer_id);
CREATE INDEX IF NOT EXISTS idx_crm_opportunities_created_at ON crm_opportunities(created_at);
