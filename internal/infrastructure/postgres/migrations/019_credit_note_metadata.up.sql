-- 019 - Campos adicionales para Notas Crédito y referencias de factura original

ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS document_type VARCHAR(20),
    ADD COLUMN IF NOT EXISTS original_invoice_id UUID,
    ADD COLUMN IF NOT EXISTS original_invoice_number VARCHAR(100),
    ADD COLUMN IF NOT EXISTS original_invoice_cufe VARCHAR(255),
    ADD COLUMN IF NOT EXISTS original_invoice_issue_on TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS discrepancy_code VARCHAR(2),
    ADD COLUMN IF NOT EXISTS discrepancy_reason TEXT;

