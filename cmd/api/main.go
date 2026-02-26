package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	appanalytics "github.com/jhoicas/Inventario-api/internal/application/analytics"
	"github.com/jhoicas/Inventario-api/internal/application/auth"
	"github.com/jhoicas/Inventario-api/internal/application/billing"
	"github.com/jhoicas/Inventario-api/internal/application/inventory"
	"github.com/jhoicas/Inventario-api/internal/application/usecase"
	infraai "github.com/jhoicas/Inventario-api/internal/infrastructure/ai"
	infradian "github.com/jhoicas/Inventario-api/internal/infrastructure/dian"
	"github.com/jhoicas/Inventario-api/internal/infrastructure/dian/signer"
	infrapdf "github.com/jhoicas/Inventario-api/internal/infrastructure/pdf"
	"github.com/jhoicas/Inventario-api/internal/infrastructure/postgres"
	httpRouter "github.com/jhoicas/Inventario-api/internal/interfaces/http"
	"github.com/jhoicas/Inventario-api/pkg/config"
	"github.com/jhoicas/Inventario-api/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("cargar configuración: " + err.Error())
	}

	log := logger.New(logger.Config{
		Env:   cfg.App.Env,
		Level: "info",
	})
	log.Info().
		Str("env", cfg.App.Env).
		Str("app", cfg.App.Name).
		Msg("iniciando aplicación")

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DB)
	if err != nil {
		log.Fatal().Err(err).Msg("conexión a PostgreSQL")
	}
	defer pool.Close()

	companyRepo := postgres.NewCompanyRepository(pool)
	userRepo := postgres.NewUserRepository(pool)
	warehouseRepo := postgres.NewWarehouseRepository(pool)
	productRepo := postgres.NewProductRepository(pool)
	customerRepo := postgres.NewCustomerRepository(pool)
	invoiceRepo := postgres.NewInvoiceRepository(pool)
	txRunner := postgres.NewTxRunner(pool)
	registerMovementUC := inventory.NewRegisterMovementUseCase(txRunner, productRepo, warehouseRepo)
	customerUC := billing.NewCustomerUseCase(customerRepo)

	xmlBuilder := infradian.NewXMLBuilderService()
	signerSvc := signer.NewDigitalSignatureService()
	dianCfg := billing.DIANConfig{
		TechnicalKey: cfg.DIAN.TechnicalKey,
		Environment:  cfg.DIAN.Environment,
		AppEnv:       cfg.DIAN.AppEnv,
		CertPath:     cfg.DIAN.CertPath,
		CertKeyPath:  cfg.DIAN.CertKeyPath,
		CertPassword: cfg.DIAN.CertPassword,
	}

	// Cliente SOAP DIAN — solo se usa si AppEnv es "test" o "prod".
	// En modo "dev" el orquestador no lo invoca.
	var dianSubmitter infradian.DIANSubmitter
	if cfg.DIAN.AppEnv != "dev" && cfg.DIAN.AppEnv != "" {
		dianSubmitter = infradian.NewSOAPDIANClient()
	}

	// DIANOrchestrator: ciclo CUFE → XML → XAdES-EPES → ZIP → Envío SOAP → Update DB
	resolutionRepo := postgres.NewBillingResolutionRepository(pool)
	dianOrchestrator := billing.NewDIANOrchestrator(
		invoiceRepo, companyRepo, customerRepo, productRepo,
		resolutionRepo, xmlBuilder, signerSvc, dianSubmitter, dianCfg,
	)

	createInvoiceUC := billing.NewCreateInvoiceUseCase(
		txRunner, registerMovementUC,
		customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo,
		dianOrchestrator, dianCfg,
	)

	analyticsRepo := postgres.NewAnalyticsRepository(pool)
	levelRepo := postgres.NewInventoryLevelRepository(pool)
	companyUC := usecase.NewCompanyUseCase(companyRepo)
	warehouseUC := usecase.NewWarehouseUseCase(warehouseRepo)
	productUC := usecase.NewProductUseCase(productRepo)
	moduleSvc := usecase.NewModuleService(companyRepo)
	analyticsUC := usecase.NewAnalyticsUseCase(analyticsRepo)
	replenishmentUC := inventory.NewReplenishmentUseCase(levelRepo, analyticsRepo)
	dashboardUC := appanalytics.NewDashboardUseCase(analyticsRepo)

	anthropicSvc := infraai.NewAnthropicService(cfg.AI.AnthropicAPIKey, cfg.AI.AnthropicModel)
	aiUC := usecase.NewAIUseCase(anthropicSvc)

	// PDF: representación gráfica de la factura electrónica DIAN
	pdfGenerator := infrapdf.NewMarotoPDFGenerator()
	invoicePDFUC := billing.NewPDFUseCase(
		invoiceRepo, companyRepo, customerRepo, productRepo, pdfGenerator,
	)
	authUC := auth.NewAuthUseCase(userRepo, companyRepo, auth.JWTConfig{
		Secret:     cfg.JWT.Secret,
		ExpMinutes: cfg.JWT.Expiration,
		Issuer:     cfg.JWT.Issuer,
	})

	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
		IdleTimeout:  time.Second * 60,
	})
	app.Use(recover.New())

	// Swagger UI en local: http://localhost:<port>/docs
	app.Use(swagger.New(swagger.Config{
		BasePath: "/",
		FilePath: "./docs/swagger.json",
		Path:     "docs",
		Title:    "Invorya API",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": cfg.App.Name})
	})

	httpRouter.Router(app, httpRouter.RouterDeps{
		CompanyUC:        companyUC,
		WarehouseUC:      warehouseUC,
		ProductUC:        productUC,
		RegisterMovement: registerMovementUC,
		Replenishment:    replenishmentUC,
		CustomerUC:       customerUC,
		CreateInvoice:    createInvoiceUC,
		InvoicePDF:       invoicePDFUC,
		AuthUC:           authUC,
		ModuleService:    moduleSvc,
		AnalyticsUC:      analyticsUC,
		DashboardUC:      dashboardUC,
		AIUC:             aiUC,
		JWTSecret:        cfg.JWT.Secret,
	})

	go func() {
		if err := app.Listen(cfg.HTTP.Addr()); err != nil {
			log.Error().Err(err).Msg("servidor HTTP finalizado")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("señal de apagado recibida, cerrando servidor...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("apagado del servidor")
	}

	log.Info().Msg("aplicación detenida")
}
