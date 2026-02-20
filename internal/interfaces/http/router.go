package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/tu-usuario/inventory-pro/internal/application/auth"
	"github.com/tu-usuario/inventory-pro/internal/application/billing"
	"github.com/tu-usuario/inventory-pro/internal/application/inventory"
	"github.com/tu-usuario/inventory-pro/internal/application/usecase"
)

// RouterDeps dependencias para el router.
type RouterDeps struct {
	CompanyUC        *usecase.CompanyUseCase
	WarehouseUC      *usecase.WarehouseUseCase
	ProductUC        *usecase.ProductUseCase
	RegisterMovement *inventory.RegisterMovementUseCase
	CustomerUC       *billing.CustomerUseCase
	CreateInvoice    *billing.CreateInvoiceUseCase
	AuthUC           *auth.AuthUseCase
	JWTSecret        string
}

// Router registra las rutas de la API.
func Router(app *fiber.App, deps RouterDeps) {
	api := app.Group("/api")

	// Auth (público)
	authGroup := api.Group("/auth")
	authHandler := NewAuthHandler(deps.AuthUC)
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)

	// Companies (público por ahora; se puede proteger con AuthMiddleware(deps.JWTSecret))
	companies := api.Group("/companies")
	companyHandler := NewCompanyHandler(deps.CompanyUC)
	companies.Get("/", companyHandler.List)
	companies.Post("/", companyHandler.Create)
	companies.Get("/:id", companyHandler.GetByID)

	// Rutas protegidas (requieren Bearer Token)
	protected := api.Group("/", AuthMiddleware(deps.JWTSecret))

	// Warehouses (protegido)
	warehouses := protected.Group("/warehouses")
	warehouseHandler := NewWarehouseHandler(deps.WarehouseUC)
	warehouses.Post("/", warehouseHandler.Create)
	warehouses.Get("/", warehouseHandler.List)
	warehouses.Get("/:id", warehouseHandler.GetByID)

	// Products (protegido)
	products := protected.Group("/products")
	productHandler := NewProductHandler(deps.ProductUC)
	products.Post("/", productHandler.Create)
	products.Get("/", productHandler.List)
	products.Get("/:id", productHandler.GetByID)
	products.Put("/:id", productHandler.Update)

	// Inventory movements (protegido)
	invGroup := protected.Group("/inventory")
	inventoryHandler := NewInventoryHandler(deps.RegisterMovement)
	invGroup.Post("/movements", inventoryHandler.RegisterMovement)

	// Customers (protegido, facturación)
	customers := protected.Group("/customers")
	customerHandler := NewCustomerHandler(deps.CustomerUC)
	customers.Post("/", customerHandler.Create)
	customers.Get("/", customerHandler.List)

	// Invoices (protegido)
	invoices := protected.Group("/invoices")
	invoiceHandler := NewInvoiceHandler(deps.CreateInvoice)
	invoices.Post("/", invoiceHandler.Create)
	invoices.Get("/:id", invoiceHandler.GetByID)
}
