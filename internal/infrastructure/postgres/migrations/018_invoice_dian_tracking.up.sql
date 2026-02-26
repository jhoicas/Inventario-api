-- Añade seguimiento del envío al WS DIAN (TrackID y errores de rechazo).
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS track_id_dian VARCHAR(255),
    ADD COLUMN IF NOT EXISTS dian_errors   TEXT;

-- Ampliar el CHECK de dian_status con los nuevos estados.
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_dian_status_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_dian_status_check
    CHECK (dian_status IN (
        'DRAFT', 'Pending', 'SIGNED', 'Sent',
        'EXITOSO', 'RECHAZADO', 'Error', 'ERROR_GENERATION'
    ));
