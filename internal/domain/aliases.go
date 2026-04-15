package domain

import "github.com/jhoicas/Inventario-api/internal/domain/entity"

// CampaignRecipient se expone en el paquete domain por compatibilidad con firmas de repositorio.
type CampaignRecipient = entity.CampaignRecipient

// Automation se expone en el paquete domain por compatibilidad con el motor de automatizaciones CRM.
type Automation = entity.CRMAutomation

// AutomationType se expone en el paquete domain por compatibilidad con el motor de automatizaciones CRM.
type AutomationType = entity.CRMAutomationType

const (
	AutomationTypeBirthday   AutomationType = entity.CRMAutomationTypeBirthday
	AutomationTypeRepurchase AutomationType = entity.CRMAutomationTypeRepurchase
)
