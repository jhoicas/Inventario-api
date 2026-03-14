package entity

import "time"

// TicketSentiment resultado del análisis de sentimiento en un ticket PQR.
type TicketSentiment string

const (
	TicketSentimentPositive TicketSentiment = "positive"
	TicketSentimentNeutral  TicketSentiment = "neutral"
	TicketSentimentNegative TicketSentiment = "negative"
)

// Estado de un ticket PQR.
const (
	TicketStatusOpen      = "open"
	TicketStatusResolved  = "resolved"
	TicketStatusClosed    = "closed"
	TicketStatusEscalated = "ESCALATED"
	TicketStatusOverdue   = "OVERDUE"
)

// CRMTicket representa un caso PQR (peticiones, quejas, reclamos) con análisis de sentimiento.
type CRMTicket struct {
	ID               string
	CompanyID        string
	CustomerID       string
	Subject          string
	Description      string
	Status           string
	Sentiment        string // positive, neutral, negative (nullable)
	EscalationReason string // razón de escalamiento (nullable)
	CreatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
