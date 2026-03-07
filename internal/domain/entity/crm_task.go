package entity

import "time"

// TaskStatus estado de una tarea CRM.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusDone      TaskStatus = "done"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// CRMTask representa una tarea manual o de seguimiento.
type CRMTask struct {
	ID          string
	CompanyID   string
	CustomerID  string // opcional
	Title       string
	Description string
	DueAt       time.Time
	Status      TaskStatus
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
