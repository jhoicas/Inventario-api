package crm

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// TaskUseCase gestión de tareas CRM.
type TaskUseCase struct {
	taskRepo repository.CRMTaskRepository
}

// NewTaskUseCase construye el caso de uso.
func NewTaskUseCase(taskRepo repository.CRMTaskRepository) *TaskUseCase {
	return &TaskUseCase{taskRepo: taskRepo}
}

// Create crea una tarea.
func (uc *TaskUseCase) Create(ctx context.Context, companyID, userID string, in dto.CreateTaskRequest) (*dto.TaskResponse, error) {
	if in.Title == "" {
		return nil, domain.ErrInvalidInput
	}
	var dueAt time.Time
	if in.DueAt != nil {
		dueAt = *in.DueAt
	}
	task := &entity.CRMTask{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		CustomerID:  in.CustomerID,
		Title:       in.Title,
		Description: in.Description,
		DueAt:       dueAt,
		Status:      entity.TaskStatusPending,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := uc.taskRepo.Create(task); err != nil {
		return nil, err
	}
	return toTaskResponse(task), nil
}

// GetByID obtiene una tarea por ID.
func (uc *TaskUseCase) GetByID(ctx context.Context, companyID, id string) (*dto.TaskResponse, error) {
	task, err := uc.taskRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domain.ErrNotFound
	}
	if task.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	return toTaskResponse(task), nil
}

// Update actualiza una tarea.
func (uc *TaskUseCase) Update(ctx context.Context, companyID, id string, in dto.UpdateTaskRequest) (*dto.TaskResponse, error) {
	task, err := uc.taskRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domain.ErrNotFound
	}
	if task.CompanyID != companyID {
		return nil, domain.ErrForbidden
	}
	if in.Title != nil {
		task.Title = *in.Title
	}
	if in.Description != nil {
		task.Description = *in.Description
	}
	if in.DueAt != nil {
		task.DueAt = *in.DueAt
	}
	if in.Status != nil {
		s := *in.Status
		if s != string(entity.TaskStatusPending) && s != string(entity.TaskStatusDone) && s != string(entity.TaskStatusCancelled) {
			return nil, domain.ErrInvalidInput
		}
		task.Status = entity.TaskStatus(s)
	}
	task.UpdatedAt = time.Now()
	if err := uc.taskRepo.Update(task); err != nil {
		return nil, err
	}
	return toTaskResponse(task), nil
}

// ListByCompany lista tareas de la empresa con filtro opcional por status.
func (uc *TaskUseCase) ListByCompany(ctx context.Context, companyID, status string, limit, offset int) (*dto.TaskResponseList, error) {
	list, err := uc.taskRepo.ListByCompany(companyID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]dto.TaskResponse, 0, len(list))
	for _, t := range list {
		items = append(items, *toTaskResponse(t))
	}
	return &dto.TaskResponseList{Items: items, Limit: limit, Offset: offset}, nil
}

// GenerateReorderAlerts devuelve sugerencias de tareas de recompra. Sin integración con inventario retorna lista vacía.
func (uc *TaskUseCase) GenerateReorderAlerts(ctx context.Context, companyID string) ([]dto.TaskAlert, error) {
	return nil, nil
}

func toTaskResponse(t *entity.CRMTask) *dto.TaskResponse {
	resp := &dto.TaskResponse{
		ID:          t.ID,
		CompanyID:   t.CompanyID,
		CustomerID:  t.CustomerID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		CreatedBy:   t.CreatedBy,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	if !t.DueAt.IsZero() {
		resp.DueAt = &t.DueAt
	}
	return resp
}
