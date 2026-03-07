package entity

import "time"

// TicketSentiment resultado del análisis de sentimiento en un ticket PQR.
type TicketSentiment string

const (
	TicketSentimentPositive TicketSentiment = "positive"
	TicketSentimentNeutral  TicketSentiment = "neutral"
	TicketSentimentNegative TicketSentiment = "negative"
)

// CRMTicket representa un caso PQR (peticiones, quejas, reclamos) con análisis de sentimiento.
type CRMTicket struct {
	ID          string
	CompanyID   string
	CustomerID  string
	Subject     string
	Description string
	Status      string
	Sentiment   string // positive, neutral, negative (nullable)
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
