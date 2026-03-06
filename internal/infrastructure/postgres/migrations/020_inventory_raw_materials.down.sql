-- Reversión 020 - Materias primas, recetas y costos adicionales en facturas

ALTER TABLE invoices
    DROP COLUMN IF EXISTS logistics_cost,
    DROP COLUMN IF EXISTS discount_total;

DROP TABLE IF EXISTS bill_of_materials;
DROP TABLE IF EXISTS raw_materials;

