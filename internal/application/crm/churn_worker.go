package crm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/jhoicas/Inventario-api/pkg/logger"
)

// ChurnWorker ejecuta prediccion de abandono y crea tareas de retencion asistidas por IA.
type ChurnWorker struct {
	analyticsRepo  repository.AIAnalyticsRepository
	taskRepo       repository.CRMTaskRepository
	salesAssistant RetentionPitchGenerator
	log            *logger.Logger
	loc            *time.Location
	daysThreshold  int
}

// RetentionPitchGenerator define el contrato para generar mensajes de retencion.
type RetentionPitchGenerator interface {
	GenerateRetentionPitch(ctx context.Context, customerName, favoriteProduct string, daysInactive int) (string, error)
}

// NewChurnWorker construye el worker de churn con zona horaria de Colombia.
func NewChurnWorker(
	analyticsRepo repository.AIAnalyticsRepository,
	taskRepo repository.CRMTaskRepository,
	salesAssistant RetentionPitchGenerator,
	log *logger.Logger,
	daysThreshold int,
) (*ChurnWorker, error) {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		return nil, fmt.Errorf("cargar zona horaria America/Bogota: %w", err)
	}
	if daysThreshold <= 0 {
		daysThreshold = 60
	}

	return &ChurnWorker{
		analyticsRepo:  analyticsRepo,
		taskRepo:       taskRepo,
		salesAssistant: salesAssistant,
		log:            log,
		loc:            loc,
		daysThreshold:  daysThreshold,
	}, nil
}

// Start ejecuta el job diariamente a las 4:00 AM en hora Colombia.
func (w *ChurnWorker) Start(ctx context.Context) {
	if w == nil || w.analyticsRepo == nil || w.taskRepo == nil || w.salesAssistant == nil {
		return
	}

	if w.log != nil {
		w.log.Info().
			Str("timezone", w.loc.String()).
			Int("days_threshold", w.daysThreshold).
			Msg("worker de churn iniciado")
	}

	for {
		now := time.Now().In(w.loc)
		nextRun := nextChurnRun(now, w.loc)
		wait := time.Until(nextRun)
		if wait < 0 {
			wait = 0
		}

		if w.log != nil {
			w.log.Info().Time("next_run", nextRun).Dur("wait", wait).Msg("churn prediction programado")
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			if w.log != nil {
				w.log.Info().Msg("worker de churn detenido")
			}
			return
		case <-timer.C:
		}

		w.RunChurnPredictionJob(ctx)
	}
}

// RunChurnPredictionJob ejecuta el ciclo completo: deteccion -> pitch IA -> tarea CRM.
func (w *ChurnWorker) RunChurnPredictionJob(ctx context.Context) {
	if w == nil {
		return
	}

	if w.log != nil {
		w.log.Info().Int("days_threshold", w.daysThreshold).Msg("churn prediction: inicio")
	}

	atRiskCustomers, err := w.analyticsRepo.GetCustomersAtRiskOfChurn(ctx, w.daysThreshold)
	if err != nil {
		if w.log != nil {
			w.log.Error().Err(err).Int("days_threshold", w.daysThreshold).Msg("churn prediction: error consultando clientes en riesgo")
		}
		return
	}

	createdTasks := 0
	for _, customer := range atRiskCustomers {
		if customer == nil {
			continue
		}

		const titlePrefix = "Alerta Abandono"
		existsToday, err := w.taskRepo.CheckTaskExistsForToday(ctx, customer.CompanyID, customer.CustomerName, titlePrefix)
		if err != nil {
			if w.log != nil {
				w.log.Error().
					Err(err).
					Str("company_id", customer.CompanyID).
					Str("customer_email", customer.CustomerEmail).
					Msg("churn prediction: error verificando deduplicacion")
			}
			continue
		}
		if existsToday {
			if w.log != nil {
				w.log.Info().
					Str("company_id", customer.CompanyID).
					Str("customer_email", customer.CustomerEmail).
					Msg("churn prediction: tarea existente para hoy, cliente omitido")
			}
			continue
		}

		pitch, err := w.salesAssistant.GenerateRetentionPitch(
			ctx,
			customer.CustomerName,
			customer.FavoriteProduct,
			customer.DaysInactive,
		)
		if err != nil {
			if w.log != nil {
				w.log.Error().
					Err(err).
					Str("company_id", customer.CompanyID).
					Str("customer_email", customer.CustomerEmail).
					Msg("churn prediction: error generando pitch")
			}
			continue
		}

		now := time.Now()
		task := &entity.CRMTask{
			ID:          uuid.NewString(),
			CompanyID:   customer.CompanyID,
			Title:       fmt.Sprintf("Alerta Abandono: %s", customer.CustomerName),
			Description: pitch,
			Status:      entity.TaskStatusPending,
			CreatedBy:   "system",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := w.taskRepo.Create(task); err != nil {
			if w.log != nil {
				w.log.Error().
					Err(err).
					Str("company_id", customer.CompanyID).
					Str("customer_email", customer.CustomerEmail).
					Msg("churn prediction: error creando tarea CRM")
			}
			continue
		}

		createdTasks++
	}

	if w.log != nil {
		w.log.Info().
			Int("customers_at_risk", len(atRiskCustomers)).
			Int("tasks_created", createdTasks).
			Int("days_threshold", w.daysThreshold).
			Msg("churn prediction: fin")
	}
}

func nextChurnRun(now time.Time, loc *time.Location) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), 4, 0, 0, 0, loc)
	if !now.Before(next) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
