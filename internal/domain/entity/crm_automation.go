package entity

import (
	"encoding/json"
	"time"
)

// CRMAutomationType enum de automatizaciones CRM.
type CRMAutomationType string

const (
	CRMAutomationTypeBirthday   CRMAutomationType = "BIRTHDAY"
	CRMAutomationTypeRepurchase CRMAutomationType = "REPURCHASE"
)

// CRMAutomation representa una automatización CRM programable.
type CRMAutomation struct {
	ID           string
	CompanyID    string
	Name         string
	Type         CRMAutomationType
	TemplateID   *string
	ScheduleCron *string
	Config       json.RawMessage
	IsActive     bool
	LastRunAt    *time.Time
}
