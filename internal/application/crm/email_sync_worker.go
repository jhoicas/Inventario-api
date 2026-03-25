package crm

import (
	"context"
	"time"
)

// EmailSyncWorker sincroniza periódicamente correos IMAP de todas las cuentas activas.
type EmailSyncWorker struct {
	emailUC  *EmailUseCase
	interval time.Duration
}

func NewEmailSyncWorker(emailUC *EmailUseCase, interval time.Duration) *EmailSyncWorker {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &EmailSyncWorker{emailUC: emailUC, interval: interval}
}

func (w *EmailSyncWorker) Start(ctx context.Context) {
	if w.emailUC == nil {
		return
	}
	w.emailUC.SyncActiveAccounts(ctx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.emailUC.SyncActiveAccounts(ctx)
		}
	}
}
