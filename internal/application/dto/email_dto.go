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

type CreateTicketFromEmailResponse struct {
	TicketID string `json:"ticket_id"`
	Status   string `json:"status"`
}
