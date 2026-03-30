- [ ] Extender DTOs de companies/company_modules para CRUD y toggles
- [ ] Extender CompanyRepository con operaciones CRUD de company_modules
- [ ] Implementar lógica de Update/Delete company y CRUD company_modules en CompanyUseCase
- [ ] Implementar queries PostgreSQL para nuevas operaciones de company_modules
- [ ] Exponer endpoints HTTP nuevos en CompanyHandler (companies + company_modules)
- [ ] Agregar endpoint alias para asignación de permisos role_screens en RBACHandler
- [ ] Crear middleware combinado de acceso (módulo activo + permisos de pantalla)
- [ ] Integrar middleware/endpoints en router
- [ ] Ejecutar validación de compilación/tests y ajustar
- [ ] Actualizar TODO.md marcando completado

- [ ] [CRM Analytics] Extender contrato de repositorio CRM con métodos de KPIs, segmentación y evolución mensual
- [ ] [CRM Analytics] Implementar consultas PostgreSQL filtradas por company_id
- [ ] [CRM Analytics] Crear DTOs de respuesta para dashboard CRM
- [ ] [CRM Analytics] Agregar handlers GET en CRMHandler bajo /api/crm/analytics
- [ ] [CRM Analytics] Registrar rutas en router
- [ ] [CRM Analytics] Ejecutar validación de compilación/tests

- [ ] [CRM Import/Bulk] Agregar DTOs para importación y envío masivo de campañas
- [ ] [CRM Import/Bulk] Extender contratos de repositorio/use case para importación Excel y cola de envío masivo
- [ ] [CRM Import/Bulk] Implementar lógica de importación Excel (excelize) y upsert de perfiles CRM
- [ ] [CRM Import/Bulk] Implementar creación de borrador de campaña + encolado bulk
- [ ] [CRM Import/Bulk] Agregar handlers HTTP POST /api/crm/import y POST /api/crm/campaigns/send-bulk
- [ ] [CRM Import/Bulk] Registrar rutas en router
- [ ] [CRM Import/Bulk] Ejecutar validación de compilación/tests

Estado actual de pruebas:
- No se ha ejecutado ninguna prueba todavía.
