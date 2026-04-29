package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/jhoicas/Inventario-api/internal/application/crm"
	infraai "github.com/jhoicas/Inventario-api/internal/infrastructure/ai"
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
	taskRepo := postgres.NewCRMTaskRepository(pool)
	aiAnalyticsRepo := postgres.NewAIAnalyticsRepository(pool)
	anthropicSvc := infraai.NewAnthropicService(cfg.AI.AnthropicAPIKey, cfg.AI.AnthropicModel)
	salesAssistant := crm.NewAISalesAssistant(anthropicSvc, logg)
	automationUC := crm.NewAutomationUseCase(
		automationRepo,
		campaignRepo,
		templateRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		anthropicSvc,
		nil,
		nil,
		logg,
		nil,
	)

	automationWorker, err := crm.NewAutomationCronWorker(automationUC, logg)
	if err != nil {
		logg.Fatal().Err(err).Msg("configurar worker de automatizaciones")
	}

	churnWorker, err := crm.NewChurnWorker(aiAnalyticsRepo, taskRepo, salesAssistant, logg, 60)
	if err != nil {
		logg.Fatal().Err(err).Msg("configurar worker de churn")
	}

	logg.Info().Msg("workers de automatizaciones y churn listos")
	go automationWorker.Start(ctx)
	go churnWorker.Start(ctx)

	<-ctx.Done()
	logg.Info().Msg("workers detenidos")
}
