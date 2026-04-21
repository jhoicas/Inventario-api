package ai

// TextToSQLSchemaDescription esquema simplificado expuesto al modelo para generar SELECTs.
// Mantener alineado con las tablas reales de PostgreSQL.
const TextToSQLSchemaDescription = `
customers:
  - id (uuid)
  - company_id (uuid)
  - name (text)
  - email (text)
  - phone (text)
  - tax_id (text)
  - birth_date (date)

crm_customer_profiles:
  - customer_id (uuid) — FK a customers.id
  - company_id (uuid)
  - category_id (uuid) — segmentación / categoría CRM
  - ltv (numeric)
  - metadata (jsonb)

crm_categories:
  - id (uuid)
  - company_id (uuid)
  - name (text)

crm_sales_hub:
  - id (uuid)
  - company_id (uuid)
  - sale_date (timestamptz)
  - items (jsonb) — detalle de líneas u otros datos embebidos
  - total_amount (numeric)

crm_products_hub:
  - id (uuid)
  - company_id (uuid)
  - product_name (text)
  - product_code (text)
  - category_id (uuid, nullable)

PARA OBTENER VENTAS POR CATEGORIA, DEBES SUMAR EL TOTAL DE crm_sales_hub, HACER UN JOIN CON crm_products_hub (PARA SABER QUE PRODUCTO SE VENDIO) Y LUEGO UN JOIN CON crm_categories (USANDO EL category_id DEL PRODUCTO). NO USES COLUMNAS QUE NO EXISTAN.

PRECAUCION CON COLUMNAS JSONB: SI VAS A CONSULTAR O EXTRAER DATOS DE UNA COLUMNA JSONB QUE ES UN ARREGLO (ARRAY), DEBES USAR jsonb_array_elements() O jsonb_array_elements_text(). NUNCA USES jsonb_object_keys() SOBRE UN ARREGLO.
`
