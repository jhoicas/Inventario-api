package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// MetaWhatsAppProvider implementa MessageProvider usando Meta Cloud API oficial.
type MetaWhatsAppProvider struct {
	accessToken   string
	phoneNumberID string
}

// NewMetaWhatsAppProvider crea un nuevo proveedor de WhatsApp.
func NewMetaWhatsAppProvider(accessToken, phoneNumberID string) *MetaWhatsAppProvider {
	return &MetaWhatsAppProvider{
		accessToken:   accessToken,
		phoneNumberID: phoneNumberID,
	}
}

func (p *MetaWhatsAppProvider) Send(ctx context.Context, to string, content string) error {
	// Meta Cloud API requiere solo dígitos (sin +, espacios ni guiones).
	cleanTo := strings.ReplaceAll(to, "+", "")
	cleanTo = strings.ReplaceAll(cleanTo, " ", "")
	cleanTo = strings.ReplaceAll(cleanTo, "-", "")

	url := fmt.Sprintf("https://graph.facebook.com/v17.0/%s/messages", p.phoneNumberID)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                cleanTo,
		"type":              "text",
		"text": map[string]string{
			"body": content,
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error realizando petición a Meta API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("respuesta de error de Meta API: %d", resp.StatusCode)
	}

	return nil
}

func (p *MetaWhatsAppProvider) ProviderType() string {
	return "WHATSAPP"
}
