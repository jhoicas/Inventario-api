-- 028_idx_crm_interactions.up.sql
-- Índice compuesto para las consultas ListInteractions (filtro + orden por fecha).
CREATE INDEX IF NOT EXISTS idx_crm_interactions_customer_created
    ON crm_interactions (customer_id, created_at DESC);
