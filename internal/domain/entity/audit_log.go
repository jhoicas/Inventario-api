package entity

import (
	"encoding/json"
	"time"
)

// AuditLog registra cambios de entidades del ERP para trazabilidad.
type AuditLog struct {
	ID         string          `json:"id"`
	CompanyID  string          `json:"company_id"`
	UserID     string          `json:"user_id"`
	Action     string          `json:"action"`
	EntityName string          `json:"entity_name"`
	EntityID   string          `json:"entity_id"`
	Changes    json.RawMessage `json:"changes"`
	CreatedAt  time.Time       `json:"created_at"`
}
