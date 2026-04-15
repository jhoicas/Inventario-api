package crm

import (
	"context"
	"fmt"
	"time"

	"github.com/jhoicas/Inventario-api/pkg/logger"
)

// AutomationCronWorker ejecuta RunDailyAutomations todos los días a las 2:00 AM en hora Bogotá.
type AutomationCronWorker struct {
	uc  *AutomationUseCase
	log *logger.Logger
	loc *time.Location
}

// NewAutomationCronWorker crea el worker con zona horaria explícita de Colombia.
func NewAutomationCronWorker(uc *AutomationUseCase, log *logger.Logger) (*AutomationCronWorker, error) {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		return nil, fmt.Errorf("cargar zona horaria America/Bogota: %w", err)
	}
	return &AutomationCronWorker{uc: uc, log: log, loc: loc}, nil
}

// Start bloquea hasta que el contexto sea cancelado.
func (w *AutomationCronWorker) Start(ctx context.Context) {
	if w == nil || w.uc == nil {
		return
	}

	if w.log != nil {
		w.log.Info().Str("timezone", w.loc.String()).Msg("worker de automatizaciones iniciado")
	}

	for {
		now := time.Now().In(w.loc)
		nextRun := nextDailyRun(now, w.loc)
		wait := time.Until(nextRun)
		if wait < 0 {
			wait = 0
		}

		if w.log != nil {
			w.log.Info().Time("next_run", nextRun).Dur("wait", wait).Msg("automatizaciones programadas para ejecución")
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			if w.log != nil {
				w.log.Info().Msg("worker de automatizaciones detenido")
			}
			return
		case <-timer.C:
		}

		w.runOnce(ctx)
	}
}

func (w *AutomationCronWorker) runOnce(ctx context.Context) {
	if w.log != nil {
		w.log.Info().Msg("automatizaciones diarias: inicio")
	}

	if err := w.uc.RunDailyAutomations(ctx); err != nil {
		if w.log != nil {
			w.log.Error().Err(err).Msg("automatizaciones diarias: error")
		}
		return
	}

	if w.log != nil {
		w.log.Info().Msg("automatizaciones diarias: fin")
	}
}

func nextDailyRun(now time.Time, loc *time.Location) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, loc)
	if !now.Before(next) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
