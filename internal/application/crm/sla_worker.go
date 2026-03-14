package crm

import (
	"context"
	"time"

	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// SLAWorker revisa periódicamente los tickets activos y los marca como OVERDUE
// si su tiempo de creación supera el SLA configurado por empresa.
type SLAWorker struct {
	ticketRepo repository.CRMTicketRepository
	interval   time.Duration
}

// NewSLAWorker construye el worker. interval es la frecuencia de revisión (ej. 24 * time.Hour).
func NewSLAWorker(ticketRepo repository.CRMTicketRepository, interval time.Duration) *SLAWorker {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &SLAWorker{ticketRepo: ticketRepo, interval: interval}
}

// Start ejecuta el worker en un loop hasta que ctx sea cancelado.
// Debe lanzarse como goroutine.
func (w *SLAWorker) Start(ctx context.Context) {
	// Correr una vez al arrancar, luego en el interval.
	w.run(ctx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.run(ctx)
		}
	}
}

func (w *SLAWorker) run(ctx context.Context) {
	_, _ = w.ticketRepo.MarkOverdueTickets(ctx)
}
