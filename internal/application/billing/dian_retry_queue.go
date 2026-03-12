package billing

import "sync"

// DIANRetryQueue mantiene una cola en memoria de facturas en CONTINGENCIA.
// Deduplica IDs para evitar reintentos duplicados innecesarios.
type DIANRetryQueue struct {
	mu      sync.Mutex
	items   []string
	indexed map[string]struct{}
}

func NewDIANRetryQueue(initialCapacity int) *DIANRetryQueue {
	if initialCapacity <= 0 {
		initialCapacity = 64
	}
	return &DIANRetryQueue{
		items:   make([]string, 0, initialCapacity),
		indexed: make(map[string]struct{}, initialCapacity),
	}
}

func (q *DIANRetryQueue) Enqueue(invoiceID string) {
	if invoiceID == "" {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, exists := q.indexed[invoiceID]; exists {
		return
	}
	q.items = append(q.items, invoiceID)
	q.indexed[invoiceID] = struct{}{}
}

func (q *DIANRetryQueue) Drain(max int) []string {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return nil
	}
	if max <= 0 || max > len(q.items) {
		max = len(q.items)
	}

	batch := append([]string(nil), q.items[:max]...)
	q.items = q.items[max:]
	for _, id := range batch {
		delete(q.indexed, id)
	}
	return batch
}
