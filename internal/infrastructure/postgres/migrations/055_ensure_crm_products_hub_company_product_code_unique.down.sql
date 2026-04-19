-- Quitar el índice único explícito (no elimina UNIQUE declarado en CREATE TABLE de 046 si existiera).

DROP INDEX IF EXISTS uq_crm_products_hub_company_product_code;
