# Migracion incremental de endpoints de listado

Fecha: 2026-04-08

## Compatibilidad temporal (legacy)

Se implemento compatibilidad temporal para endpoints paginados con el siguiente comportamiento:

- Formato por defecto (canonico): `{items,total,limit,offset}`.
- Modo compatibilidad bajo demanda:
	- Query param: `?legacy=true`
	- Header: `X-Pagination-Format: legacy` o `X-Legacy-Pagination: true`
- Respuesta en modo legacy: `{items,page:{total,limit,offset}}`

### Rollout y feature flags

- `PAGINATION_LEGACY_COMPAT_ENABLED` (default: `true`):
	- `true`: habilita consumo legacy opt-in.
	- `false`: fuerza siempre formato canonico.
- `PAGINATION_LEGACY_COMPAT_ENDPOINTS` (default: `*`):
	- Permite acotar compatibilidad legacy por endpoint interno.
	- Ejemplo: `email.messages.list,billing.invoices.list`

### Telemetria de consumo legacy

Cada respuesta legacy emite log estructurado con:

- `event=pagination_legacy_response_served`
- `endpoint`
- `method`
- `path`
- `company_id`
- `deprecation_date`

### Fecha sugerida de deprecacion

- Fecha objetivo de retiro de modo legacy: `2026-07-31`
- Recomendacion operacional:
	- Semana 1-2: monitorear adopcion con logs
	- Semana 3-6: notificar consumidores legacy activos
	- Semana 7-8: deshabilitar por endpoint con allowlist
	- Semana 9+: apagar bandera global

## Fase 1 - Endpoints con limit/offset

- GET /api/companies
- GET /api/admin/companies
- GET /api/customers
- GET /api/products
- GET /api/suppliers
- GET /api/warehouses
- GET /api/users
- GET /api/settings/email-accounts
- GET /api/emails
- GET /api/purchase-orders
- GET /api/inventory/movements
- GET /api/invoices
- GET /api/invoices/credit-notes
- GET /api/invoices/debit-notes
- GET /api/crm/customers
- GET /api/crm/categories
- GET /api/crm/categories/{id}/benefits
- GET /api/crm/tasks
- GET /api/crm/customers/{id}/interactions
- GET /api/crm/tickets
- GET /api/crm/opportunities
- GET /api/crm/customers/{id}/purchase-history
- GET /api/crm/campaign-templates

Exclusion explicita (legacy):
- GET /api/crm/remarketing (LEGACY_ARRAY)

## Fase 2 - Clasificacion

### Ya tienen total correcto

- /api/companies
- /api/admin/companies
- /api/customers
- /api/products
- /api/suppliers
- /api/warehouses
- /api/users
- /api/settings/email-accounts
- /api/emails
- /api/purchase-orders
- /api/inventory/movements
- /api/invoices
- /api/invoices/credit-notes
- /api/invoices/debit-notes
- /api/crm/customers
- /api/crm/categories
- /api/crm/categories/{id}/benefits
- /api/crm/tasks
- /api/crm/customers/{id}/interactions
- /api/crm/tickets
- /api/crm/opportunities
- /api/crm/customers/{id}/purchase-history
- /api/crm/campaign-templates

### No tienen total

- Ninguno dentro del set paginado activo.

### Tienen total incorrecto

- Ninguno detectado dentro del set paginado activo.

## Fase 3 - Correccion por modulo

No se requirio correccion funcional adicional de calculo de total en codigo productivo para los endpoints paginados activos.
Se mantuvo /api/crm/remarketing como LEGACY_ARRAY (fuera de contrato paginado).

## Fase 4 - Tests agregados/ajustados por endpoint

Se agregaron pruebas de invariantes de paginacion para verificar:
- total coincide con el filtro aplicado
- total no cambia al variar limit/offset

Nuevos tests:
- internal/interfaces/http/admin_user_handler_pagination_test.go
- internal/interfaces/http/supplier_handler_pagination_test.go
- internal/interfaces/http/email_handler_pagination_test.go

Test agregado en archivo existente:
- internal/interfaces/http/inventory_handler_test.go: TestInventoryHandler_GetPurchaseOrders_TotalInvariantAcrossPagination

## Changelog por endpoint (antes/despues e impacto frontend)

Formato de salida objetivo:
- Antes: contratos heterogeneos (arrays planos o sin garantia explicita de total consistente)
- Despues: { items, total, limit, offset }
- Impacto frontend: permite etiqueta estable "Mostrando X-Y de N" sin heuristicas

### Company

- Endpoint: GET /api/companies
- Antes: paginacion no estandarizada globalmente
- Despues: {items,total,limit,offset}
- Impacto frontend: paginador deterministico por total real

- Endpoint: GET /api/admin/companies
- Antes: paginacion no estandarizada globalmente
- Despues: {items,total,limit,offset}
- Impacto frontend: misma logica de tabla para vistas admin

### Customer/CRM base

- Endpoint: GET /api/customers
- Antes: paginacion no estandarizada globalmente
- Despues: {items,total,limit,offset}
- Impacto frontend: total consistente para filtros de busqueda

### Product/Supplier/Warehouse/User

- Endpoint: GET /api/products
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: rango y total confiables

- Endpoint: GET /api/suppliers
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: total por filtro search estable

- Endpoint: GET /api/warehouses
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: paginacion uniforme

- Endpoint: GET /api/users
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: UX consistente en administracion de usuarios

### Email

- Endpoint: GET /api/settings/email-accounts
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: contador total estable por bandeja de cuentas

- Endpoint: GET /api/emails
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: total robusto al filtrar por customer_id/is_read

### Inventory

- Endpoint: GET /api/purchase-orders
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: pagina actual no altera total del universo filtrado

- Endpoint: GET /api/inventory/movements
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: contador estable por filtros de producto/bodega/tipo/fechas

### Billing

- Endpoint: GET /api/invoices
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: listado principal de facturas con total confiable

- Endpoint: GET /api/invoices/credit-notes
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: mismo componente de tabla reutilizable

- Endpoint: GET /api/invoices/debit-notes
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: misma semantica de paginacion en notas

### CRM

- Endpoint: GET /api/crm/customers
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: ranking/listado CRM con total estable

- Endpoint: GET /api/crm/categories
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: paginador unificado en catalogos CRM

- Endpoint: GET /api/crm/categories/{id}/benefits
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: total consistente por categoria

- Endpoint: GET /api/crm/tasks
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: total por estado estable

- Endpoint: GET /api/crm/customers/{id}/interactions
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: total consistente por filtros type/start/end

- Endpoint: GET /api/crm/tickets
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: total estable por filtros search/status/sort

- Endpoint: GET /api/crm/opportunities
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: funnel listado paginado con total confiable

- Endpoint: GET /api/crm/customers/{id}/purchase-history
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: historial paginado con contador correcto

- Endpoint: GET /api/crm/campaign-templates
- Antes: no estandar global
- Despues: {items,total,limit,offset}
- Impacto frontend: plantillas con navegacion de paginas consistente

### Legacy explicito

- Endpoint: GET /api/crm/remarketing
- Antes: array plano
- Despues: se mantiene array plano con etiqueta LEGACY_ARRAY
- Impacto frontend: requiere manejo legacy explicito; no usar el componente paginado estandar
