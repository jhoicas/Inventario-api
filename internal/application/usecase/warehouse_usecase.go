package usecase

import (
	"time"

	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// WarehouseUseCase casos de uso CRUD para bodegas.
type WarehouseUseCase struct {
	repo repository.WarehouseRepository
}

// NewWarehouseUseCase construye el caso de uso.
func NewWarehouseUseCase(repo repository.WarehouseRepository) *WarehouseUseCase {
	return &WarehouseUseCase{repo: repo}
}

// Create crea una nueva bodega.
func (uc *WarehouseUseCase) Create(companyID string, in dto.CreateWarehouseRequest) (*dto.WarehouseResponse, error) {
	now := time.Now()
	warehouse := &entity.Warehouse{
		ID:        uuid.New().String(),
		CompanyID: companyID,
		Name:      in.Name,
		Address:   in.Address,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.repo.Create(warehouse); err != nil {
		return nil, err
	}
	return toWarehouseResponse(warehouse), nil
}

// GetByID obtiene una bodega por ID.
func (uc *WarehouseUseCase) GetByID(id string) (*dto.WarehouseResponse, error) {
	warehouse, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if warehouse == nil {
		return nil, nil
	}
	return toWarehouseResponse(warehouse), nil
}

// Update actualiza una bodega.
func (uc *WarehouseUseCase) Update(id string, in dto.UpdateWarehouseRequest) (*dto.WarehouseResponse, error) {
	warehouse, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if warehouse == nil {
		return nil, nil
	}
	if in.Name != nil {
		warehouse.Name = *in.Name
	}
	if in.Address != nil {
		warehouse.Address = *in.Address
	}
	warehouse.UpdatedAt = time.Now()
	if err := uc.repo.Update(warehouse); err != nil {
		return nil, err
	}
	return toWarehouseResponse(warehouse), nil
}

// List lista bodegas por empresa con paginación.
func (uc *WarehouseUseCase) List(companyID string, limit, offset int) (*dto.WarehouseListResponse, error) {
	list, err := uc.repo.ListByCompany(companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]dto.WarehouseResponse, 0, len(list))
	for _, w := range list {
		items = append(items, *toWarehouseResponse(w))
	}
	return &dto.WarehouseListResponse{
		Items: items,
		Page:  dto.PageResponse{Limit: limit, Offset: offset},
	}, nil
}

// Delete elimina una bodega por ID.
func (uc *WarehouseUseCase) Delete(id string) error {
	return uc.repo.Delete(id)
}

func toWarehouseResponse(w *entity.Warehouse) *dto.WarehouseResponse {
	if w == nil {
		return nil
	}
	return &dto.WarehouseResponse{
		ID:        w.ID,
		CompanyID: w.CompanyID,
		Name:      w.Name,
		Address:   w.Address,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}
