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
	"github.com/tu-usuario/inventory-pro/internal/application/auth"
	"github.com/tu-usuario/inventory-pro/internal/application/billing"
	"github.com/tu-usuario/inventory-pro/internal/application/inventory"
	"github.com/tu-usuario/inventory-pro/internal/application/usecase"
	infradian "github.com/tu-usuario/inventory-pro/internal/infrastructure/dian"
	"github.com/tu-usuario/inventory-pro/internal/infrastructure/dian/signer"
	"github.com/tu-usuario/inventory-pro/internal/infrastructure/postgres"
	httpRouter "github.com/tu-usuario/inventory-pro/internal/interfaces/http"
	"github.com/tu-usuario/inventory-pro/pkg/config"
	"github.com/tu-usuario/inventory-pro/pkg/logger"
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
		TechnicalKey:  cfg.DIAN.TechnicalKey,
		Environment:   cfg.DIAN.Environment,
		CertPath:      cfg.DIAN.CertPath,
		CertKeyPath:   cfg.DIAN.CertKeyPath,
		CertPassword:  cfg.DIAN.CertPassword,
	}
	createInvoiceUC := billing.NewCreateInvoiceUseCase(txRunner, registerMovementUC, customerRepo, companyRepo, productRepo, warehouseRepo, invoiceRepo, xmlBuilder, signerSvc, dianCfg)

	companyUC := usecase.NewCompanyUseCase(companyRepo)
	warehouseUC := usecase.NewWarehouseUseCase(warehouseRepo)
	productUC := usecase.NewProductUseCase(productRepo)
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
		CustomerUC:       customerUC,
		CreateInvoice:    createInvoiceUC,
		AuthUC:           authUC,
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
