package entity

// SLAConfig configuración de tiempo máximo de resolución por empresa y tipo de ticket.
// ticket_type vacío equivale al tipo por defecto (aplica a todos los tickets sin tipo específico).
type SLAConfig struct {
	CompanyID  string
	TicketType string // vacío = default
	MaxHours   int    // horas máximas antes de marcar como OVERDUE
}
