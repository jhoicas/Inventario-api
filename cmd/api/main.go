package main

import (
	"context"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	appanalytics "github.com/jhoicas/Inventario-api/internal/application/analytics"
	"github.com/jhoicas/Inventario-api/internal/application/auth"
	"github.com/jhoicas/Inventario-api/internal/application/billing"
	"github.com/jhoicas/Inventario-api/internal/application/crm"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/application/inventory"
	"github.com/jhoicas/Inventario-api/internal/application/usecase"
	dianws "github.com/jhoicas/Inventario-api/internal/billing"
	infraai "github.com/jhoicas/Inventario-api/internal/infrastructure/ai"
	infradian "github.com/jhoicas/Inventario-api/internal/infrastructure/dian"
	"github.com/jhoicas/Inventario-api/internal/infrastructure/dian/signer"
	inframail "github.com/jhoicas/Inventario-api/internal/infrastructure/mail"
	infrapdf "github.com/jhoicas/Inventario-api/internal/infrastructure/pdf"
	"github.com/jhoicas/Inventario-api/internal/infrastructure/postgres"
	infrasecurity "github.com/jhoicas/Inventario-api/internal/infrastructure/security"
	httpRouter "github.com/jhoicas/Inventario-api/internal/interfaces/http"
	"github.com/jhoicas/Inventario-api/pkg/config"
	"github.com/jhoicas/Inventario-api/pkg/logger"
)

// @title Tu API ERP
// @version 1.0
// @description API para el sistema ERP.
// @host api.naturerp.ludoia.com
// @BasePath /
//
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Escribe "Bearer " seguido de un espacio y tu token.
// @description Escribe "Bearer " seguido de un espacio y tu token.
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
	supplierRepo := postgres.NewSupplierRepository(pool)
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
	dianRetryQueue := billing.NewDIANRetryQueue(1024)
	dianOrchestrator.SetRetryQueue(dianRetryQueue)
	dianRetryWorker := billing.NewDIANRetryWorker(dianOrchestrator, dianRetryQueue, 15*time.Minute, 50)
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go dianRetryWorker.Start(workerCtx)

	smtpCfg := dianws.SMTPConfig{
		Host:         cfg.SMTP.Host,
		Port:         cfg.SMTP.Port,
		User:         cfg.SMTP.User,
		Password:     cfg.SMTP.Password,
		From:         cfg.SMTP.From,
		ResendAPIKey: cfg.SMTP.ResendAPIKey,
		ResendAPIURL: cfg.SMTP.ResendAPIURL,
	}

	createInvoiceUC := billing.NewCreateInvoiceUseCase(
		txRunner, registerMovementUC,
		customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo,
		dianOrchestrator, dianCfg,
	)

	createCreditNoteUC := billing.NewCreateCreditNoteUseCase(
		txRunner, registerMovementUC,
		customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo,
		dianOrchestrator, dianCfg,
	)

	createDebitNoteUC := billing.NewCreateDebitNoteUseCase(
		txRunner,
		customerRepo, companyRepo, productRepo, invoiceRepo,
		dianOrchestrator, dianCfg,
	)

	createVoidInvoiceUC := billing.NewCreateVoidInvoiceUseCase(
		txRunner,
		customerRepo, invoiceRepo,
		dianOrchestrator, dianCfg,
	)

	analyticsRepo := postgres.NewAnalyticsRepository(pool)
	levelRepo := postgres.NewInventoryLevelRepository(pool)
	stockRepo := postgres.NewStockRepository(pool)
	movementRepo := postgres.NewInventoryMovementRepository(pool)
	reorderConfigRepo := postgres.NewReorderConfigRepository(pool)
	dianSettingsRepo := postgres.NewDIANSettingsRepository(pool)
	purchaseOrderRepo := postgres.NewPurchaseOrderRepository(pool)
	companyUC := usecase.NewCompanyUseCase(companyRepo, resolutionRepo)
	warehouseUC := usecase.NewWarehouseUseCase(warehouseRepo)
	productUC := usecase.NewProductUseCase(productRepo)
	supplierUC := usecase.NewSupplierUseCase(supplierRepo)
	purchaseOrderUC := inventory.NewPurchaseOrderUseCase(purchaseOrderRepo, supplierRepo, warehouseRepo, txRunner, registerMovementUC)
	updateReorderConfigUC := inventory.NewUpdateReorderConfigUseCase(productRepo, reorderConfigRepo)
	encryptor, err := infrasecurity.NewAesGCMEncryptor(cfg.JWT.Secret)
	if err != nil {
		log.Fatal().Err(err).Msg("configurar cifrado para DIAN settings")
	}
	certStore := infrasecurity.NewDIANCertificateFileStore(cfg.DIAN.CertStoragePath)
	dianSettingsUC := usecase.NewDIANSettingsUseCase(companyRepo, dianSettingsRepo, certStore, encryptor)
	moduleSvc := usecase.NewModuleService(companyRepo)
	analyticsUC := usecase.NewAnalyticsUseCase(analyticsRepo)
	rawMaterialAnalyticsUC := usecase.NewRawMaterialAnalyticsUseCase(analyticsRepo)
	replenishmentUC := inventory.NewReplenishmentUseCase(levelRepo, analyticsRepo)
	getStockUC := inventory.NewGetStockUseCase(stockRepo)
	listMovementsUC := inventory.NewGetMovementsUseCase(movementRepo)
	dashboardUC := appanalytics.NewDashboardUseCase(analyticsRepo)

	anthropicSvc := infraai.NewAnthropicService(cfg.AI.AnthropicAPIKey, cfg.AI.AnthropicModel)
	aiUC := usecase.NewAIUseCase(anthropicSvc)

	// CRM: repositorios y casos de uso (módulo crm)
	crmCategoryRepo := postgres.NewCRMCategoryRepository(pool)
	crmBenefitRepo := postgres.NewCRMBenefitRepository(pool)
	crmProfileRepo := postgres.NewCRMProfileRepository(pool)
	crmInteractionRepo := postgres.NewCRMInteractionRepository(pool)
	crmTaskRepo := postgres.NewCRMTaskRepository(pool)
	crmTicketRepo := postgres.NewCRMTicketRepository(pool)
	crmCampaignRepo := postgres.NewCRMCampaignRepository(pool)
	crmTemplateRepo := postgres.NewCRMCampaignTemplateRepository(pool)
	crmOpportunityRepo := postgres.NewCRMOpportunityRepository(pool)
	slaConfigRepo := postgres.NewSLAConfigRepository(pool)
	_ = slaConfigRepo // disponible para futuros endpoints
	slaWorker := crm.NewSLAWorker(crmTicketRepo, 24*time.Hour)
	go slaWorker.Start(workerCtx)
	loyaltyUC := crm.NewLoyaltyUseCase(crmProfileRepo, customerRepo, crmCategoryRepo, crmBenefitRepo, crmInteractionRepo)
	taskUC := crm.NewTaskUseCase(crmTaskRepo)
	aiCRMUC := crm.NewAICRMUseCase(anthropicSvc)
	pqrUC := crm.NewPQRUseCase(crmTicketRepo, customerRepo, aiCRMUC, crmInteractionRepo)
	mailSender, err := inframail.NewSMTPSenderFromEnv()
	if err != nil {
		log.Error().Err(err).Msg("configurar SMTPSender para campañas CRM (usando net/smtp)")
	}
	campaignUC := crm.NewCampaignUseCase(crmCampaignRepo, customerRepo, crmProfileRepo, crmInteractionRepo, mailSender)
	templateUC := crm.NewCampaignTemplateUseCase(crmTemplateRepo)
	opportunityUC := crm.NewOpportunityUseCase(crmOpportunityRepo)
	crmHandler := httpRouter.NewCRMHandler(loyaltyUC, taskUC, pqrUC, aiCRMUC, crmInteractionRepo, opportunityUC, invoiceRepo, campaignUC, templateUC)

	// Worker diario de reposición crítica → crea tareas CRM de reabastecimiento.
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		run := func() {
			ctx := context.Background()

			companies, err := companyRepo.List(1000, 0)
			if err != nil {
				log.Error().Err(err).Msg("daily_replenishment_worker: listar compañías")
				return
			}

			for _, c := range companies {
				list, err := replenishmentUC.GenerateReplenishmentList(ctx, c.ID, "")
				if err != nil {
					log.Error().Err(err).
						Str("company_id", c.ID).
						Msg("daily_replenishment_worker: GenerateReplenishmentList")
					continue
				}

				for _, item := range list {
					if item.Priority != 1 {
						continue
					}

					hasOpen, err := taskUC.HasOpenReplenishmentTask(ctx, c.ID, item.ProductName)
					if err != nil {
						log.Error().Err(err).
							Str("company_id", c.ID).
							Str("product_id", item.ProductID).
							Msg("daily_replenishment_worker: HasOpenReplenishmentTask")
						continue
					}
					if hasOpen {
						continue
					}

					leadDays := 3
					due := time.Now().AddDate(0, 0, leadDays)
					dueAt := due

					_, err = taskUC.Create(ctx, c.ID, "system", dto.CreateTaskRequest{
						CustomerID:  "",
						Title:       "Reabastecer " + item.ProductName,
						Description: "Producto crítico en lista de reposición automática",
						DueAt:       &dueAt,
					})
					if err != nil {
						log.Error().Err(err).
							Str("company_id", c.ID).
							Str("product_id", item.ProductID).
							Msg("daily_replenishment_worker: Create task")
						continue
					}
				}
			}
		}

		// primera ejecución inmediata al iniciar la app
		run()

		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()

	// PDF: representación gráfica de la factura electrónica DIAN
	pdfGenerator := infrapdf.NewMarotoPDFGenerator()
	invoicePDFUC := billing.NewPDFUseCase(
		invoiceRepo, companyRepo, customerRepo, productRepo, pdfGenerator,
	)
	invoiceMailer := dianws.NewInvoiceMailer(
		invoiceRepo, companyRepo, customerRepo, productRepo, pdfGenerator, smtpCfg,
	)
	dianOrchestrator.SetMailer(invoiceMailer)
	authUC := auth.NewAuthUseCase(userRepo, companyRepo, auth.JWTConfig{
		Secret:     cfg.JWT.Secret,
		ExpMinutes: cfg.JWT.Expiration,
		Issuer:     cfg.JWT.Issuer,
	})

	// DIAN GetAcquirer: consulta de contribuyentes por tipo y número de documento
	customerLookupHandler := dianws.NewCustomerLookupHandler(cfg.DIAN.AppEnv)

	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
		IdleTimeout:  time.Second * 60,
	})
	app.Use(fiberlogger.New())
	app.Use(recover.New())

	// CORS dinámico (Fail-Fast): orígenes desde variable de entorno
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		stdlog.Fatal("ERROR CRÍTICO: La variable de entorno ALLOWED_ORIGINS no está configurada o está vacía. El servidor no puede iniciar de forma segura.")
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: true,
	}))

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
		CompanyUC:              companyUC,
		CompanyRepo:            companyRepo,
		WarehouseUC:            warehouseUC,
		ProductUC:              productUC,
		SupplierUC:             supplierUC,
		UserRepo:               userRepo,
		RegisterMovement:       registerMovementUC,
		Replenishment:          replenishmentUC,
		GetStock:               getStockUC,
		ListMovements:          listMovementsUC,
		ReorderConfig:          updateReorderConfigUC,
		DIANSettingsUC:         dianSettingsUC,
		PurchaseOrder:          purchaseOrderUC,
		CustomerUC:             customerUC,
		CreateInvoice:          createInvoiceUC,
		ReturnInvoice:          createCreditNoteUC,
		DebitNote:              createDebitNoteUC,
		VoidInvoice:            createVoidInvoiceUC,
		InvoicePDF:             invoicePDFUC,
		AuthUC:                 authUC,
		ModuleService:          moduleSvc,
		AnalyticsUC:            analyticsUC,
		RawMaterialAnalyticsUC: rawMaterialAnalyticsUC,
		DashboardUC:            dashboardUC,
		AIUC:                   aiUC,
		CRMHandler:             crmHandler,
		CustomerLookup:         customerLookupHandler,
		InvoiceMailer:          invoiceMailer,
		JWTSecret:              cfg.JWT.Secret,
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
	workerCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("apagado del servidor")
	}

	log.Info().Msg("aplicación detenida")
}
