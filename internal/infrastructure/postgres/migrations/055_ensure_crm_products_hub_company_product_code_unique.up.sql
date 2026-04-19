-- Garantiza índice único (company_id, product_code) para INSERT ... ON CONFLICT en importación CRM.
-- Elimina duplicados previos: conserva el registro más antiguo (created_at, luego id) y repunta crm_sale_items_hub.

DO $$
BEGIN
    -- 1) Repuntar líneas de venta hacia el producto canónico de cada (company_id, product_code)
    WITH ranked AS (
        SELECT
            id,
            first_value(id) OVER (
                PARTITION BY company_id, product_code
                ORDER BY created_at ASC NULLS LAST, id ASC
            ) AS keeper_id
        FROM crm_products_hub
    )
    UPDATE crm_sale_items_hub sih
    SET product_id = r.keeper_id
    FROM ranked r
    WHERE sih.product_id = r.id
      AND r.id <> r.keeper_id;

    -- 2) Borrar filas duplicadas en crm_products_hub (mantener keeper por partición)
    DELETE FROM crm_products_hub ph
    WHERE ph.id IN (
        SELECT id
        FROM (
            SELECT
                id,
                row_number() OVER (
                    PARTITION BY company_id, product_code
                    ORDER BY created_at ASC NULLS LAST, id ASC
                ) AS rn
            FROM crm_products_hub
        ) sub
        WHERE sub.rn > 1
    );
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_crm_products_hub_company_product_code
    ON crm_products_hub (company_id, product_code);
