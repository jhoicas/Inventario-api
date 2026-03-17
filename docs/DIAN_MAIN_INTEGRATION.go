//go:build ignore
// +build ignore

// main.go integration example

package main

import (
	// ... otros imports ...

	"github.com/gofiber/fiber/v2"
	dianws "github.com/jhoicas/Inventario-api/internal/billing"
	"github.com/jhoicas/Inventario-api/internal/infrastructure/postgres"
	"github.com/jhoicas/Inventario-api/internal/interfaces/http"
	"github.com/jhoicas/Inventario-api/pkg/config"
)

func setupRouter(app *fiber.App, cfg *config.Config) {
	// ... Inicializar repositorios y servicios ...

	// 1. Crear CompanyRepository (si no existe)
	companyRepo := postgres.NewCompanyRepository(db)

	// 2. Crear CustomerLookupHandler
	customerLookup := dianws.NewCustomerLookupHandler(cfg.DIAN.AppEnv)

	// 3. Preparar RouterDeps incluyendo CompanyRepo
	deps := http.RouterDeps{
		CompanyUC:   companyUseCase,
		CompanyRepo: companyRepo, // ← NUEVO: requerido para DIANConfigMiddleware
		WarehouseUC: warehouseService,
		ProductUC:   productService,
		UserRepo:    userRepository,
		// ... resto de dependencias ...
		CustomerLookup: customerLookup,
		JWTSecret:      cfg.JWT.Secret,
	}

	// 4. Registrar rutas con middleware automático
	http.Router(app, deps)
	// → DIANConfigMiddleware se registrará automáticamente en /api/customers/lookup
}

// Nota: El middleware DIANConfigMiddleware ahora:
// 1. Lee company_id del JWT (via AuthMiddleware)
// 2. Consulta CompanyRepository.GetByID(companyID)
// 3. Llama company.DianConfig() para obtener URL y cert
// 4. Inyecta valores en c.Locals("dian_url") y c.Locals("dian_cert")
// 5. CustomerLookupHandler.Lookup() usa estos valores en GetAcquirer()
