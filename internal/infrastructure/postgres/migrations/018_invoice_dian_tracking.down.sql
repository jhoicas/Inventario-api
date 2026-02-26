ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_dian_status_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_dian_status_check
    CHECK (dian_status IN ('Pending', 'Sent', 'Error', 'DRAFT', 'SIGNED', 'ERROR_GENERATION'));

ALTER TABLE invoices DROP COLUMN IF EXISTS dian_errors;
ALTER TABLE invoices DROP COLUMN IF EXISTS track_id_dian;
