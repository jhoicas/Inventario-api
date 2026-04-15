package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/jhoicas/Inventario-api/internal/application/crm"
	"github.com/jhoicas/Inventario-api/internal/infrastructure/postgres"
	"github.com/jhoicas/Inventario-api/pkg/config"
	"github.com/jhoicas/Inventario-api/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("cargar configuración: %v", err)
	}

	logg := logger.New(logger.Config{Env: cfg.App.Env, Level: "info"})
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DB)
	if err != nil {
		logg.Fatal().Err(err).Msg("conexión a PostgreSQL")
	}
	defer pool.Close()

	automationRepo := postgres.NewCRMAutomationRepository(pool)
	campaignRepo := postgres.NewCRMCampaignRepository(pool)
	templateRepo := postgres.NewCRMCampaignTemplateRepository(pool)
	automationUC := crm.NewAutomationUseCase(automationRepo, campaignRepo, templateRepo, nil, logg)

	worker, err := crm.NewAutomationCronWorker(automationUC, logg)
	if err != nil {
		logg.Fatal().Err(err).Msg("configurar worker de automatizaciones")
	}

	logg.Info().Msg("worker de automatizaciones listo")
	worker.Start(ctx)
	logg.Info().Msg("worker de automatizaciones finalizado")
}
