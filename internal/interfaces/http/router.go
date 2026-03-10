package http

import (
	"github.com/gofiber/fiber/v2"
	appanalytics "github.com/jhoicas/Inventario-api/internal/application/analytics"
	"github.com/jhoicas/Inventario-api/internal/application/auth"
	"github.com/jhoicas/Inventario-api/internal/application/billing"
	"github.com/jhoicas/Inventario-api/internal/application/inventory"
	"github.com/jhoicas/Inventario-api/internal/application/usecase"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// RouterDeps dependencias para el router.
type RouterDeps struct {
	CompanyUC        *usecase.CompanyUseCase
	WarehouseUC      *usecase.WarehouseUseCase
	ProductUC        *usecase.ProductUseCase
	RegisterMovement *inventory.RegisterMovementUseCase
	Replenishment    *inventory.ReplenishmentUseCase
	CustomerUC       *billing.CustomerUseCase
	CreateInvoice    *billing.CreateInvoiceUseCase
	ReturnInvoice    *billing.CreateCreditNoteUseCase
	InvoicePDF       *billing.PDFUseCase
	AuthUC           *auth.AuthUseCase
	ModuleService    *usecase.ModuleService
	AnalyticsUC             *usecase.AnalyticsUseCase
	RawMaterialAnalyticsUC   *usecase.RawMaterialAnalyticsUseCase
	DashboardUC      *appanalytics.DashboardUseCase
	AIUC             *usecase.AIUseCase
	CRMHandler       *CRMHandler
	JWTSecret        string
}

// Router registra todas las rutas de la API aplicando:
//  1. AuthMiddleware  — valida JWT y carga user_id, company_id y role en el contexto.
//  2. RequireModule   — verifica que la empresa tenga activo el módulo SaaS requerido.
//  3. RequireRole     — verifica que el rol del usuario tenga permiso para la operación.
//
// Tabla de roles:
//
//	admin     → acceso total
//	bodeguero → inventario, productos (escritura/lectura), IA
//	vendedor  → facturación (emitir/consultar), clientes, productos (lectura)
func Router(app *fiber.App, deps RouterDeps) {
	api := app.Group("/api")

	// ── Rutas públicas ─────────────────────────────────────────────────────────
	authHandler := NewAuthHandler(deps.AuthUC)
	api.Post("/auth/register", authHandler.Register)
	api.Post("/auth/login", authHandler.Login)

	// Companies — público (onboarding de nuevos tenants)
	companyHandler := NewCompanyHandler(deps.CompanyUC)
	api.Get("/companies", companyHandler.List)
	api.Post("/companies", companyHandler.Create)
	api.Get("/companies/:id", companyHandler.GetByID)

	// ── Rutas protegidas (JWT obligatorio) ────────────────────────────────────
	// Todos los grupos siguientes heredan AuthMiddleware.
	protected := api.Group("/", AuthMiddleware(deps.JWTSecret))

	// ── Catálogos de lectura (JWT solo — todos los roles pueden leer para armar la UI) ──

	warehouseHandler := NewWarehouseHandler(deps.WarehouseUC)
	wh := protected.Group("/warehouses")
	wh.Get("/", warehouseHandler.List)
	wh.Get("/:id", warehouseHandler.GetByID)
	// Creación de bodegas: solo admin
	wh.Post("/", RequireRole(entity.RoleAdmin), warehouseHandler.Create)

	productHandler := NewProductHandler(deps.ProductUC)
	prod := protected.Group("/products")
	prod.Get("/", productHandler.List)
	prod.Get("/:id", productHandler.GetByID)
	// Escritura de productos: admin y bodeguero
	prod.Post("/", RequireRole(entity.RoleAdmin, entity.RoleBodeguero), productHandler.Create)
	prod.Put("/:id", RequireRole(entity.RoleAdmin, entity.RoleBodeguero), productHandler.Update)

	customerHandler := NewCustomerHandler(deps.CustomerUC)
	cust := protected.Group("/customers")
	cust.Get("/", customerHandler.List)
	// Creación de clientes: admin y vendedor
	cust.Post("/", RequireRole(entity.RoleAdmin, entity.RoleVendedor), customerHandler.Create)

	// ── Inventario (módulo 'inventory' + roles) ────────────────────────────────
	inventoryHandler := NewInventoryHandler(deps.RegisterMovement, deps.Replenishment)
	invGroup := protected.Group("/inventory", RequireModule(entity.ModuleInventory, deps.ModuleService))

	// POST /inventory/movements — admin y bodeguero
	invGroup.Post("/movements",
		RequireRole(entity.RoleAdmin, entity.RoleBodeguero),
		inventoryHandler.RegisterMovement,
	)
	// GET /inventory/replenishment-list — admin y bodeguero
	invGroup.Get("/replenishment-list",
		RequireRole(entity.RoleAdmin, entity.RoleBodeguero),
		inventoryHandler.GetReplenishmentList,
	)

	// ── Facturación (módulo 'billing' + roles) ─────────────────────────────────
	invoiceHandler := NewInvoiceHandler(deps.CreateInvoice, deps.ReturnInvoice, deps.InvoicePDF)
	invGroup2 := protected.Group("/invoices", RequireModule(entity.ModuleBilling, deps.ModuleService))

	// POST — emitir factura: admin y vendedor
	invGroup2.Post("/",
		RequireRole(entity.RoleAdmin, entity.RoleVendedor),
		invoiceHandler.Create,
	)
	// POST — registrar devolución (Nota Crédito): admin y vendedor
	invGroup2.Post("/:id/return",
		RequireRole(entity.RoleAdmin, entity.RoleVendedor),
		invoiceHandler.HandleReturn,
	)
	// GET (consultas y descarga) — todos los roles con billing activo
	invGroup2.Get("/:id/status", invoiceHandler.GetDIANStatus)
	invGroup2.Get("/:id/pdf", invoiceHandler.DownloadPDF)
	invGroup2.Get("/:id", invoiceHandler.GetByID)

	// ── Analytics (módulo 'analytics' + solo admin) ────────────────────────────
	analyticsHandler := NewAnalyticsHandler(deps.AnalyticsUC, deps.RawMaterialAnalyticsUC)
	analyticsGroup := protected.Group("/analytics",
		RequireModule(entity.ModuleAnalytics, deps.ModuleService),
		RequireRole(entity.RoleAdmin),
	)
		analyticsGroup.Get("/margins", analyticsHandler.GetMargins)
		analyticsGroup.Get("/raw-materials-impact", analyticsHandler.GetRawMaterialImpactRanking)

	// ── Dashboard (JWT + solo admin) ───────────────────────────────────────────
	dashboardHandler := NewDashboardHandler(deps.DashboardUC)
	dashboardGroup := protected.Group("/dashboard", RequireRole(entity.RoleAdmin))
	dashboardGroup.Get("/summary", dashboardHandler.GetSummary)

	// ── CRM (módulo 'crm') ─────────────────────────────────────────────────────
	if deps.CRMHandler != nil {
		crmGroup := protected.Group("/crm", RequireModule(entity.ModuleCRM, deps.ModuleService))
		h := deps.CRMHandler
		crmGroup.Get("/customers/:id/profile360", h.GetProfile360)
		crmGroup.Put("/customers/:id/category", h.AssignCategory)
		crmGroup.Get("/categories", h.ListCategories)
		crmGroup.Get("/categories/:id/benefits", h.ListBenefitsByCategory)
		// Beneficios: escritura solo admin
		crmGroup.Post("/categories/:categoryId/benefits", RequireRole(entity.RoleAdmin), h.CreateBenefit)
		crmGroup.Put("/benefits/:benefitId", RequireRole(entity.RoleAdmin), h.UpdateBenefit)
		crmGroup.Post("/tasks", h.CreateTask)
		crmGroup.Get("/tasks", h.ListTasks)
		crmGroup.Get("/tasks/:id", h.GetTask)
		crmGroup.Put("/tasks/:id", h.UpdateTask)
		crmGroup.Post("/interactions", h.CreateInteraction)
		crmGroup.Post("/tickets", h.CreateTicket)
		crmGroup.Get("/tickets", h.ListTickets)
		crmGroup.Get("/tickets/:id", h.GetTicket)
		crmGroup.Put("/tickets/:id", h.UpdateTicket)
		crmGroup.Post("/ai/campaign-copy", h.GenerateCampaignCopy)
		crmGroup.Post("/ai/summarize-timeline", h.SummarizeTimeline)
	}

	// ── IA (reservado para futuros usos; sugerencia de clasificación de productos deshabilitada — parametrización manual)
	// aiHandler := NewAIHandler(deps.AIUC)
	// aiGroup := protected.Group("/ai", RequireRole(entity.RoleAdmin, entity.RoleBodeguero))
	// Ruta eliminada: aiGroup.Post("/suggest-classification", aiHandler.SuggestClassification)
}
