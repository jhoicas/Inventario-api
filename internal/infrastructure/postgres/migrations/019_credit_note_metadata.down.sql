-- Reversión 019 - Campos adicionales para Notas Crédito

ALTER TABLE invoices
    DROP COLUMN IF EXISTS document_type,
    DROP COLUMN IF EXISTS original_invoice_id,
    DROP COLUMN IF EXISTS original_invoice_number,
    DROP COLUMN IF EXISTS original_invoice_cufe,
    DROP COLUMN IF EXISTS original_invoice_issue_on,
    DROP COLUMN IF EXISTS discrepancy_code,
    DROP COLUMN IF EXISTS discrepancy_reason;

