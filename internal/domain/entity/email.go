package entity

import "time"

// EmailAccount representa una cuenta IMAP por compañía (tenant).
type EmailAccount struct {
	ID           string
	CompanyID    string
	EmailAddress string
	IMAPServer   string
	IMAPPort     int
	Password     string // almacenada cifrada
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Email representa un correo sincronizado desde IMAP.
type Email struct {
	ID          string
	AccountID   string
	CompanyID   string
	MessageID   string
	CustomerID  string
	FromAddress string
	ToAddress   string
	Subject     string
	BodyHTML    string
	BodyText    string
	ReceivedAt  time.Time
	IsRead      bool
	CreatedAt   time.Time
	Attachments []EmailAttachment
}

// EmailAttachment representa un adjunto asociado a un correo.
type EmailAttachment struct {
	ID       string
	EmailID  string
	FileName string
	FileURL  string
	MIMEType string
	Size     int
}
