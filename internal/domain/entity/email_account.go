package entity

import "time"

// EmailAccountConfig representa la configuración híbrida de cuentas de correo
// (OAuth para Google/Microsoft y credenciales tradicionales para IMAP/SMTP).
type EmailAccountConfig struct {
	ID           string
	UserID       string
	CompanyID    string
	Provider     string // google | microsoft | custom
	EmailAddress string
	AccessToken  string
	RefreshToken string
	ImapHost     string
	ImapPort     int
	SmtpHost     string
	SmtpPort     int
	AppPassword  string // almacenada cifrada
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
