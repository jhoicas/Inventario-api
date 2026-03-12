package billing

import (
	"context"
	"log"
	"time"
)

// DIANRetryWorker procesa la cola de facturas en CONTINGENCIA cada intervalo.
type DIANRetryWorker struct {
	orchestrator *DIANOrchestrator
	queue        *DIANRetryQueue
	interval     time.Duration
	batchSize    int
}

func NewDIANRetryWorker(orchestrator *DIANOrchestrator, queue *DIANRetryQueue, interval time.Duration, batchSize int) *DIANRetryWorker {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	if batchSize <= 0 {
		batchSize = 50
	}
	return &DIANRetryWorker{
		orchestrator: orchestrator,
		queue:        queue,
		interval:     interval,
		batchSize:    batchSize,
	}
}

func (w *DIANRetryWorker) Start(ctx context.Context) {
	if w.orchestrator == nil || w.queue == nil {
		return
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce()
		}
	}
}

func (w *DIANRetryWorker) runOnce() {
	ids := w.queue.Drain(w.batchSize)
	if len(ids) == 0 {
		return
	}

	log.Printf("[DIAN][WORKER] reintentando %d factura(s) en contingencia", len(ids))
	for _, invoiceID := range ids {
		w.orchestrator.RetryAsync(invoiceID)
	}
}
