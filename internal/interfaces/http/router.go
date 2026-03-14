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

	supplierHandler := NewSupplierHandler(deps.SupplierUC)
	sup := protected.Group("/suppliers")
	sup.Get("/", supplierHandler.List)
	sup.Get("/:id", supplierHandler.GetByID)
	sup.Post("/", RequireRole(entity.RoleAdmin, entity.RoleBodeguero), supplierHandler.Create)
	sup.Put("/:id", RequireRole(entity.RoleAdmin, entity.RoleBodeguero), supplierHandler.Update)

	customerHandler := NewCustomerHandler(deps.CustomerUC)
	cust := protected.Group("/customers")
	cust.Get("/", customerHandler.List)
	// Creación de clientes: admin y vendedor
	cust.Post("/", RequireRole(entity.RoleAdmin, entity.RoleVendedor), customerHandler.Create)
	// Consulta DIAN por documento: JWT + RequireModule(billing) + DIANConfigMiddleware
	if deps.CustomerLookup != nil && deps.CompanyRepo != nil {
		cust.Get("/lookup",
			RequireModule(entity.ModuleBilling, deps.ModuleService),
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

	// ── Inventario (módulo 'inventory' + roles) ────────────────────────────────
	inventoryHandler := NewInventoryHandler(deps.RegisterMovement, deps.Replenishment, deps.GetStock, deps.ListMovements, deps.ReorderConfig, deps.Stocktake, deps.PurchaseOrder)
	po := protected.Group("/purchase-orders", RequireModule(entity.ModuleInventory, deps.ModuleService))
	po.Post("/",
		RequireRole(entity.RoleAdmin, entity.RoleBodeguero),
		inventoryHandler.CreatePurchaseOrder,
	)
	po.Put("/:id/receive",
		RequireRole(entity.RoleAdmin, entity.RoleBodeguero),
		inventoryHandler.ReceivePurchaseOrder,
	)

	prod.Put("/:id/reorder-config",
		RequireModule(entity.ModuleInventory, deps.ModuleService),
		RequireRole(entity.RoleAdmin, entity.RoleBodeguero),
		inventoryHandler.UpdateReorderConfig,
	)
	invGroup := protected.Group("/inventory", RequireModule(entity.ModuleInventory, deps.ModuleService))
	// GET /inventory/movements — admin y bodeguero
	invGroup.Get("/movements",
		RequireRole(entity.RoleAdmin, entity.RoleBodeguero),
		inventoryHandler.ListMovements,
	)

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
	// GET /inventory/stock — admin y bodeguero
	invGroup.Get("/stock",
		RequireRole(entity.RoleAdmin, entity.RoleBodeguero),
		inventoryHandler.GetStock,
	)
	// Stocktake (conteo físico) — admin y bodeguero
	invGroup.Post("/stocktake",
		RequireRole(entity.RoleAdmin, entity.RoleBodeguero),
		inventoryHandler.CreateStocktakeSnapshot,
	)
	invGroup.Put("/stocktake/:id",
		RequireRole(entity.RoleAdmin, entity.RoleBodeguero),
		inventoryHandler.UpdateStocktakeCounts,
	)
	invGroup.Post("/stocktake/:id/close",
		RequireRole(entity.RoleAdmin, entity.RoleBodeguero),
		inventoryHandler.CloseStocktake,
	)

	// ── Facturación (módulo 'billing' + roles) ─────────────────────────────────
	invoiceHandler := NewInvoiceHandlerWithBillingOps(deps.CreateInvoice, deps.ReturnInvoice, deps.DebitNote, deps.VoidInvoice, deps.InvoicePDF, deps.InvoiceMailer)
	invGroup2 := protected.Group("/invoices", RequireModule(entity.ModuleBilling, deps.ModuleService))

	// GET — listar facturas con filtros y paginación: admin y vendedor
	invGroup2.Get("/",
		RequireRole(entity.RoleAdmin, entity.RoleVendedor),
		invoiceHandler.GetInvoices,
	)
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
	// POST — registrar Nota Débito: admin y vendedor
	invGroup2.Post("/:id/debit-note",
		RequireRole(entity.RoleAdmin, entity.RoleVendedor),
		invoiceHandler.HandleDebitNote,
	)
	// POST — anular factura (Nota Crédito total): admin y vendedor
	invGroup2.Post("/:id/void",
		RequireRole(entity.RoleAdmin, entity.RoleVendedor),
		invoiceHandler.HandleVoidInvoice,
	)
	// POST — enviar factura por correo al cliente: admin y vendedor
	invGroup2.Post("/:id/send-email",
		RequireRole(entity.RoleAdmin, entity.RoleVendedor),
		invoiceHandler.SendEmail,
	)
	// POST — reintentar envío DIAN de factura en contingencia: admin y vendedor
	invGroup2.Post("/:id/retry-dian",
		RequireRole(entity.RoleAdmin, entity.RoleVendedor),
		invoiceHandler.RetryDIAN,
	)
	// GET (consultas y descarga) — todos los roles con billing activo
	invGroup2.Get("/:id/status", invoiceHandler.GetDIANStatus)
	invGroup2.Get("/:id/pdf", invoiceHandler.DownloadPDF)
	invGroup2.Get("/:id", invoiceHandler.GetByID)

	// ── Correos manuales (módulo 'billing' + roles) ───────────────────────────
	emailGroup := protected.Group("/emails", RequireModule(entity.ModuleBilling, deps.ModuleService))
	emailGroup.Post("/send",
		RequireRole(entity.RoleAdmin, entity.RoleVendedor),
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
		crmGroup := protected.Group("/crm", RequireModule(entity.ModuleCRM, deps.ModuleService))
		h := deps.CRMHandler
		crmGroup.Get("/customers/:id/profile360", h.GetProfile360)
		crmGroup.Put("/customers/:id/category", h.AssignCategory)
		crmGroup.Post("/customers/:id/points/award", h.AwardPoints)
		crmGroup.Get("/customers/:id/points/balance", h.GetLoyaltyBalance)
		crmGroup.Post("/customers/:id/points/redeem", h.RedeemPoints)
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
		crmGroup.Get("/customers/:id/interactions", h.ListInteractions)
		crmGroup.Post("/tickets", h.CreateTicket)
		crmGroup.Get("/tickets", h.ListTickets)
		crmGroup.Get("/tickets/overdue", h.ListOverdueTickets)
		crmGroup.Get("/tickets/:id", h.GetTicket)
		crmGroup.Put("/tickets/:id", h.UpdateTicket)
		crmGroup.Put("/tickets/:id/escalate", h.EscalateTicket)
		crmGroup.Post("/ai/campaign-copy", h.GenerateCampaignCopy)
		crmGroup.Post("/ai/summarize-timeline", h.SummarizeTimeline)
		// Opportunities (embudo de ventas)
		crmGroup.Post("/opportunities", h.CreateOpportunity)
		crmGroup.Put("/opportunities/:id/stage", h.UpdateOpportunityStage)
		crmGroup.Get("/opportunities/funnel", h.GetOpportunityFunnel)
		// Campaigns
		crmGroup.Post("/campaigns", RequireRole(entity.RoleAdmin), h.CreateCampaign)
		crmGroup.Get("/campaigns/:id/metrics", h.GetCampaignMetrics)
		// Historial de compras: requiere módulo billing activo además de crm
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
