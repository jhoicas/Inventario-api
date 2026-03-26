package dto

import "time"

type CreateEmailAccountRequest struct {
	EmailAddress string `json:"email_address"`
	IMAPServer   string `json:"imap_server"`
	IMAPPort     int    `json:"imap_port"`
	Password     string `json:"password"`
	IsActive     *bool  `json:"is_active,omitempty"`
}

type UpdateEmailAccountRequest struct {
	EmailAddress *string `json:"email_address,omitempty"`
	IMAPServer   *string `json:"imap_server,omitempty"`
	IMAPPort     *int    `json:"imap_port,omitempty"`
	Password     *string `json:"password,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}

type EmailAccountResponse struct {
	ID           string    `json:"id"`
	CompanyID    string    `json:"company_id"`
	EmailAddress string    `json:"email_address"`
	IMAPServer   string    `json:"imap_server"`
	IMAPPort     int       `json:"imap_port"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TestIMAPConnectionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type EmailAttachmentResponse struct {
	ID       string `json:"id"`
	FileName string `json:"file_name"`
	FileURL  string `json:"file_url"`
	MIMEType string `json:"mime_type"`
	Size     int    `json:"size"`
}

type EmailResponse struct {
	ID          string                    `json:"id"`
	AccountID   string                    `json:"account_id"`
	CustomerID  string                    `json:"customer_id,omitempty"`
	MessageID   string                    `json:"message_id"`
	FromAddress string                    `json:"from_address"`
	ToAddress   string                    `json:"to_address"`
	Subject     string                    `json:"subject"`
	BodyHTML    string                    `json:"body_html"`
	BodyText    string                    `json:"body_text"`
	ReceivedAt  time.Time                 `json:"received_at"`
	IsRead      bool                      `json:"is_read"`
	CreatedAt   time.Time                 `json:"created_at"`
	Attachments []EmailAttachmentResponse `json:"attachments,omitempty"`
}

type EmailListResponse struct {
	Items  []EmailResponse `json:"items"`
	Total  int64           `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type EmailMessage struct {
	ID      string    `json:"id"`
	Subject string    `json:"subject"`
	From    string    `json:"from"`
	Date    time.Time `json:"date"`
	Snippet string    `json:"snippet"`
	IsRead  bool      `json:"is_read"`
}

type AccountEmailListResponse struct {
	Provider string         `json:"provider"`
	Items    []EmailMessage `json:"items"`
}

type CreateTicketFromEmailResponse struct {
	TicketID string `json:"ticket_id"`
	Status   string `json:"status"`
}

type OAuthEmailAccountRequest struct {
	Provider     string `json:"provider"`
	AuthCode     string `json:"auth_code"`
	RedirectURI  string `json:"redirect_uri"`
	EmailAddress string `json:"email_address"`
	IsActive     *bool  `json:"is_active,omitempty"`
}

type GoogleOAuthCredentialRequest struct {
	Credential   string `json:"credential"`
	EmailAddress string `json:"email_address"`
	IsActive     *bool  `json:"is_active,omitempty"`
}

type GoogleOAuthCredentialStatusResponse struct {
	Configured   bool   `json:"configured"`
	Provider     string `json:"provider,omitempty"`
	EmailAddress string `json:"email_address,omitempty"`
	IsActive     bool   `json:"is_active,omitempty"`
}

type CustomEmailAccountRequest struct {
	EmailAddress string `json:"email_address"`
	ImapHost     string `json:"imap_host"`
	ImapPort     int    `json:"imap_port"`
	SmtpHost     string `json:"smtp_host"`
	SmtpPort     int    `json:"smtp_port"`
	AppPassword  string `json:"app_password"`
	IsActive     *bool  `json:"is_active,omitempty"`
}

type EmailAccountConfigResponse struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	CompanyID    string    `json:"company_id"`
	Provider     string    `json:"provider"`
	EmailAddress string    `json:"email_address"`
	ImapHost     string    `json:"imap_host,omitempty"`
	ImapPort     int       `json:"imap_port,omitempty"`
	SmtpHost     string    `json:"smtp_host,omitempty"`
	SmtpPort     int       `json:"smtp_port,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
