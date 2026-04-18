package repository

import "context"

// MessageProvider define el contrato para cualquier proveedor de mensajería (SMS, WhatsApp, Email, etc).
type MessageProvider interface {
	// Send envía un mensaje al destinatario (to) con el contenido dado.
	Send(ctx context.Context, to string, content string) error
	
	// ProviderType devuelve el tipo de canal que maneja ("EMAIL", "SMS", "WHATSAPP").
	ProviderType() string
}
