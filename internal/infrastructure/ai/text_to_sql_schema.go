package ai

// TextToSQLSchemaDescription esquema simplificado expuesto al modelo para generar SELECTs.
// Mantener alineado con las tablas reales de PostgreSQL.
const TextToSQLSchemaDescription = `
DICCIONARIO DE TERMINOS:
- "Segmento" o "Categoria de cliente" => tabla crm_categories (categorias de clientes).
- Relacion cliente-categoria => customers -> crm_customer_profiles -> crm_categories (via category_id).
- "Categoria de producto" => tabla crm_category_product_hub. USARLA SOLO SI EL USUARIO DICE EXPLICITAMENTE "categoria de producto".

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
  - customer_id (uuid) — FK a customers.id
  - sale_date (timestamptz)
  - items (jsonb) — detalle de líneas u otros datos embebidos
  - total_amount (numeric)

crm_products_hub:
  - id (uuid)
  - company_id (uuid)
  - product_name (text)
  - product_code (text)
  - category_id (uuid, nullable)

crm_category_product_hub:
  - id (uuid)
  - company_id (uuid)
  - name (text)

REPORTE "VENTAS POR CATEGORIA" (SEGMENTO DE CLIENTE) EN DASHBOARD:
- JOIN OBLIGATORIO: crm_sales_hub -> customers (por customer_id) -> crm_customer_profiles (por customer_id) -> LEFT JOIN crm_categories (por category_id).
- AGRUPAR SEGMENTOS CON: COALESCE(crm_categories.name, 'SIN_SEGMENTO').
- CALCULO DE VENTAS: SUM(crm_sales_hub.total_amount).
- NO USES COLUMNAS QUE NO EXISTAN.
- FILTRADO POR EMPRESA: DEBES APLICAR company_id EN TODAS LAS TABLAS INVOLUCRADAS.

PRECAUCION CON COLUMNAS JSONB: SI VAS A CONSULTAR O EXTRAER DATOS DE UNA COLUMNA JSONB QUE ES UN ARREGLO (ARRAY), DEBES USAR jsonb_array_elements() O jsonb_array_elements_text(). NUNCA USES jsonb_object_keys() SOBRE UN ARREGLO.
`
