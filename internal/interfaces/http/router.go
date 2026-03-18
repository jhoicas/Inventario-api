package http

import (
	"github.com/gofiber/fiber/v2"
	appanalytics "github.com/jhoicas/Inventario-api/internal/application/analytics"
	"github.com/jhoicas/Inventario-api/internal/application/auth"
	"github.com/jhoicas/Inventario-api/internal/application/billing"
	"github.com/jhoicas/Inventario-api/internal/application/inventory"
	"github.com/jhoicas/Inventario-api/internal/application/usecase"
	dianws "github.com/jhoicas/Inventario-api/internal/billing"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// RouterDeps dependencias para el router.
type RouterDeps struct {
	CompanyUC              *usecase.CompanyUseCase
	CompanyRepo            repository.CompanyRepository // Para inyectar configuración DIAN
	WarehouseUC            *usecase.WarehouseUseCase
	ProductUC              *usecase.ProductUseCase
	SupplierUC             *usecase.SupplierUseCase
	UserRepo               repository.UserRepository
	RegisterMovement       *inventory.RegisterMovementUseCase
	Replenishment          *inventory.ReplenishmentUseCase
	GetStock               *inventory.GetStockUseCase
	ListMovements          ListMovementsUseCase
	ReorderConfig          *inventory.UpdateReorderConfigUseCase
	DIANSettingsUC         *usecase.DIANSettingsUseCase
	Stocktake              *inventory.StocktakeUseCase
	PurchaseOrder          *inventory.PurchaseOrderUseCase
	CustomerUC             *billing.CustomerUseCase
	CreateInvoice          *billing.CreateInvoiceUseCase
	ReturnInvoice          *billing.CreateCreditNoteUseCase
	DebitNote              *billing.CreateDebitNoteUseCase
	VoidInvoice            *billing.CreateVoidInvoiceUseCase
	InvoicePDF             *billing.PDFUseCase
	AuthUC                 *auth.AuthUseCase
	ModuleService          *usecase.ModuleService
	RBACUC                 *usecase.RBACUseCase
	AnalyticsUC            *usecase.AnalyticsUseCase
	RawMaterialAnalyticsUC *usecase.RawMaterialAnalyticsUseCase
	DashboardUC            *appanalytics.DashboardUseCase
	AIUC                   *usecase.AIUseCase
	CRMHandler             *CRMHandler
	CustomerLookup         *dianws.CustomerLookupHandler
	InvoiceMailer          InvoiceMailerUseCase
	JWTSecret              string
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
	api.Post("/companies/:id/resolutions", companyHandler.CreateResolution)
	api.Get("/companies/:id/resolutions", companyHandler.ListResolutions)

	// ── Rutas protegidas (JWT obligatorio) ────────────────────────────────────
	// Todos los grupos siguientes heredan AuthMiddleware.
	protected := api.Group("/", AuthMiddleware(deps.JWTSecret))
	screenAccess := fiber.Handler(func(c *fiber.Ctx) error { return c.Next() })
	if deps.RBACUC != nil {
		screenAccess = RequirePermission(deps.RBACUC)
	}

	if deps.RBACUC != nil {
		rbacHandler := NewRBACHandler(deps.RBACUC)
		rbacGroup := protected.Group("/rbac")
		rbacGroup.Get("/menu", rbacHandler.GetCurrentMenu)
		rbacGroup.Get("/roles", RequireRole(entity.RoleAdmin), rbacHandler.ListRoles)
		rbacGroup.Get("/modules", RequireRole(entity.RoleAdmin), rbacHandler.GetCatalog)
		rbacGroup.Get("/roles/:role_id/menu", RequireRole(entity.RoleAdmin), rbacHandler.GetRoleMenu)
		rbacGroup.Put("/roles/:role_id/screens", RequireRole(entity.RoleAdmin), rbacHandler.UpdateRoleScreens)
	}

	// ── Catálogos de lectura (JWT solo — todos los roles pueden leer para armar la UI) ──

	warehouseHandler := NewWarehouseHandler(deps.WarehouseUC)
	wh := protected.Group("/warehouses", RequireModule(entity.ModuleInventory, deps.ModuleService), screenAccess)
	wh.Get("/", warehouseHandler.List)
	wh.Get("/:id", warehouseHandler.GetByID)
	wh.Post("/", warehouseHandler.Create)

	productHandler := NewProductHandler(deps.ProductUC)
	prod := protected.Group("/products", RequireModule(entity.ModuleInventory, deps.ModuleService), screenAccess)
	prod.Get("/", productHandler.List)
	prod.Get("/:id", productHandler.GetByID)
	prod.Post("/", productHandler.Create)
	prod.Put("/:id", productHandler.Update)

	supplierHandler := NewSupplierHandler(deps.SupplierUC)
	sup := protected.Group("/suppliers", RequireModule(entity.ModuleInventory, deps.ModuleService), screenAccess)
	sup.Get("/", supplierHandler.List)
	sup.Get("/:id", supplierHandler.GetByID)
	sup.Post("/", supplierHandler.Create)
	sup.Put("/:id", supplierHandler.Update)
	sup.Put("/:id/deactivate", supplierHandler.Deactivate)

	customerHandler := NewCustomerHandler(deps.CustomerUC)
	cust := protected.Group("/customers", RequireModule(entity.ModuleBilling, deps.ModuleService), screenAccess)
	cust.Get("/", customerHandler.List)
	cust.Post("/", customerHandler.Create)
	cust.Put("/:id", customerHandler.Update)
	cust.Put("/:id/deactivate", customerHandler.Deactivate)
	// Consulta DIAN por documento: JWT + RequireModule(billing) + DIANConfigMiddleware
	if deps.CustomerLookup != nil && deps.CompanyRepo != nil {
		cust.Get("/lookup",
			dianws.DIANConfigMiddleware(deps.CompanyRepo),
			deps.CustomerLookup.Lookup,
		)
	}

	// Gestión de usuarios (solo admin)
	userHandler := NewUserHandler(deps.UserRepo)
	usersGroup := protected.Group("/users", RequireRole(entity.RoleAdmin))
	usersGroup.Get("/", userHandler.List)
	usersGroup.Post("/", userHandler.Create)
	usersGroup.Put("/:id", userHandler.Update)

	resolutionsGroup := protected.Group("/resolutions",
		RequireModule(entity.ModuleBilling, deps.ModuleService),
		screenAccess,
	)
	resolutionsGroup.Get("/", companyHandler.ListMyResolutions)
	resolutionsGroup.Post("/", companyHandler.CreateMyResolution)

	if deps.DIANSettingsUC != nil {
		settingsHandler := NewSettingsHandler(deps.DIANSettingsUC)
		protected.Get("/settings/dian",
			RequireModule(entity.ModuleBilling, deps.ModuleService),
			screenAccess,
			settingsHandler.GetDIANSettings,
		)
		protected.Put("/settings/dian",
			RequireModule(entity.ModuleBilling, deps.ModuleService),
			screenAccess,
			settingsHandler.UpdateDIANSettings,
		)
		protected.Get("/dian/settings",
			RequireModule(entity.ModuleBilling, deps.ModuleService),
			screenAccess,
			settingsHandler.GetDIANSettings,
		)
		protected.Put("/dian/settings",
			RequireModule(entity.ModuleBilling, deps.ModuleService),
			screenAccess,
			settingsHandler.UpdateDIANSettings,
		)
		protected.Get("/dian/configuration",
			RequireModule(entity.ModuleBilling, deps.ModuleService),
			screenAccess,
			settingsHandler.GetDIANSettings,
		)
		protected.Put("/dian/configuration",
			RequireModule(entity.ModuleBilling, deps.ModuleService),
			screenAccess,
			settingsHandler.UpdateDIANSettings,
		)
	}

	// ── Inventario (módulo 'inventory' + roles) ────────────────────────────────
	inventoryHandler := NewInventoryHandler(deps.RegisterMovement, deps.Replenishment, deps.GetStock, deps.ListMovements, deps.ReorderConfig, deps.Stocktake, deps.PurchaseOrder)
	po := protected.Group("/purchase-orders", RequireModule(entity.ModuleInventory, deps.ModuleService), screenAccess)
	po.Get("/",
		inventoryHandler.GetPurchaseOrders,
	)
	po.Post("/",
		inventoryHandler.CreatePurchaseOrder,
	)
	po.Put("/:id/receive",
		inventoryHandler.ReceivePurchaseOrder,
	)

	prod.Put("/:id/reorder-config",
		inventoryHandler.UpdateReorderConfig,
	)
	invGroup := protected.Group("/inventory", RequireModule(entity.ModuleInventory, deps.ModuleService), screenAccess)
	invGroup.Get("/movements",
		inventoryHandler.ListMovements,
	)

	invGroup.Post("/movements",
		inventoryHandler.RegisterMovement,
	)
	invGroup.Get("/replenishment-list",
		inventoryHandler.GetReplenishmentList,
	)
	invGroup.Get("/stock",
		inventoryHandler.GetStock,
	)
	invGroup.Post("/stocktake",
		inventoryHandler.CreateStocktakeSnapshot,
	)
	invGroup.Put("/stocktake/:id",
		inventoryHandler.UpdateStocktakeCounts,
	)
	invGroup.Post("/stocktake/:id/close",
		inventoryHandler.CloseStocktake,
	)

	// ── Facturación (módulo 'billing' + roles) ─────────────────────────────────
	invoiceHandler := NewInvoiceHandlerWithBillingOps(deps.CreateInvoice, deps.ReturnInvoice, deps.DebitNote, deps.VoidInvoice, deps.InvoicePDF, deps.InvoiceMailer)
	invGroup2 := protected.Group("/invoices", RequireModule(entity.ModuleBilling, deps.ModuleService), screenAccess)

	invGroup2.Get("/",
		invoiceHandler.GetInvoices,
	)
	invGroup2.Get("/credit-notes",
		invoiceHandler.GetCreditNotes,
	)
	invGroup2.Get("/debit-notes",
		invoiceHandler.GetDebitNotes,
	)
	invGroup2.Post("/",
		invoiceHandler.Create,
	)
	invGroup2.Post("/:id/return",
		invoiceHandler.HandleReturn,
	)
	invGroup2.Post("/:id/debit-note",
		invoiceHandler.HandleDebitNote,
	)
	invGroup2.Post("/:id/void",
		invoiceHandler.HandleVoidInvoice,
	)
	invGroup2.Post("/:id/send-email",
		invoiceHandler.SendEmail,
	)
	invGroup2.Post("/:id/retry-dian",
		invoiceHandler.RetryDIAN,
	)
	invGroup2.Get("/:id/status", invoiceHandler.GetDIANStatus)
	invGroup2.Get("/:id/pdf", invoiceHandler.DownloadPDF)
	invGroup2.Get("/:id", invoiceHandler.GetByID)

	billingGroup := protected.Group("/billing", RequireModule(entity.ModuleBilling, deps.ModuleService), screenAccess)
	billingGroup.Get("/dian/summary",
		invoiceHandler.GetDIANSummary,
	)

	// ── Correos manuales (módulo 'billing' + roles) ───────────────────────────
	emailGroup := protected.Group("/emails", RequireModule(entity.ModuleBilling, deps.ModuleService), screenAccess)
	emailGroup.Post("/send",
		invoiceHandler.SendCustomEmail,
	)

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
		crmGroup := protected.Group("/crm", RequireModule(entity.ModuleCRM, deps.ModuleService), screenAccess)
		h := deps.CRMHandler
		crmGroup.Get("/customers/:id/profile360", h.GetProfile360)
		crmGroup.Put("/customers/:id/category", h.AssignCategory)
		crmGroup.Post("/loyalty/points", h.AwardPoints)
		crmGroup.Get("/customers/:id/loyalty", h.GetLoyalty)
		crmGroup.Post("/loyalty/redeem", h.RedeemPoints)
		crmGroup.Get("/categories", h.ListCategories)
		crmGroup.Post("/categories", h.CreateCategory)
		crmGroup.Put("/categories/:id", h.UpdateCategory)
		crmGroup.Put("/categories/:id/deactivate", h.DeactivateCategory)
		crmGroup.Get("/categories/:id/benefits", h.ListBenefitsByCategory)
		crmGroup.Post("/categories/:categoryId/benefits", h.CreateBenefit)
		crmGroup.Put("/benefits/:benefitId", h.UpdateBenefit)
		crmGroup.Post("/tasks", h.CreateTask)
		crmGroup.Get("/tasks", h.ListTasks)
		crmGroup.Get("/tasks/:id", h.GetTask)
		crmGroup.Put("/tasks/:id", h.UpdateTask)
		crmGroup.Post("/interactions", h.CreateInteraction)
		crmGroup.Get("/customers/:id/interactions", h.ListInteractions)
		crmGroup.Post("/tickets", h.CreateTicket)
		crmGroup.Get("/tickets", h.ListTickets)
		crmGroup.Get("/tickets/overdue", h.ListOverdueTickets)
		crmGroup.Get("/tickets/:id", h.GetTicket)
		crmGroup.Put("/tickets/:id", h.UpdateTicket)
		crmGroup.Put("/tickets/:id/escalate", h.EscalateTicket)
		crmGroup.Post("/ai/campaign-copy", h.GenerateCampaignCopy)
		crmGroup.Post("/ai/summarize-timeline", h.SummarizeTimeline)
		crmGroup.Post("/opportunities", h.CreateOpportunity)
		crmGroup.Get("/opportunities", h.ListOpportunities)
		crmGroup.Put("/opportunities/:id/stage", h.UpdateOpportunityStage)
		crmGroup.Get("/opportunities/funnel", h.GetOpportunityFunnel)
		crmGroup.Post("/campaigns", h.CreateCampaign)
		crmGroup.Get("/campaigns/:id/metrics", h.GetCampaignMetrics)
		crmGroup.Post("/campaign-templates", h.CreateCampaignTemplate)
		crmGroup.Get("/campaign-templates", h.ListCampaignTemplates)
		crmGroup.Delete("/campaign-templates/:id", h.DeleteCampaignTemplate)
		crmGroup.Get("/customers/:id/purchase-history",
			RequireModule(entity.ModuleBilling, deps.ModuleService),
			h.GetPurchaseHistory,
		)
	}

	// ── IA (reservado para futuros usos; sugerencia de clasificación de productos deshabilitada — parametrización manual)
	// aiHandler := NewAIHandler(deps.AIUC)
	// aiGroup := protected.Group("/ai", RequireRole(entity.RoleAdmin, entity.RoleBodeguero))
	// Ruta eliminada: aiGroup.Post("/suggest-classification", aiHandler.SuggestClassification)
}
