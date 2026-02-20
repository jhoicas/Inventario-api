-- Restaurar columna unit_measure (código texto) para compatibilidad con el código.
-- La app usa unit_measure; measurement_unit_id queda para catálogo DIAN futuro.
ALTER TABLE products ADD COLUMN IF NOT EXISTS unit_measure VARCHAR(10) NOT NULL DEFAULT '94';
