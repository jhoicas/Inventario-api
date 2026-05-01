package entity

import "time"

// NotificationLog registra notificaciones enviadas por cualquier canal.
type NotificationLog struct {
	ID           string
	CompanyID    string
	CustomerID   string
	CustomerName *string
	CustomerEmail *string
	CustomerPhone *string
	Type         string
	Channel      string
	Subject      string
	Body         string
	SentAt       time.Time
	Status       string
	ErrorMessage string
}
