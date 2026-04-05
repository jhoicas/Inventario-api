package repository

import (
	"context"

	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// CompanyRepository define el puerto de persistencia para Company (DIP).
// La implementación vive en infrastructure.
type CompanyRepository interface {
	Create(company *entity.Company) error
	GetByID(id string) (*entity.Company, error)
	GetByNIT(nit string) (*entity.Company, error)
	Update(company *entity.Company) error
	List(limit, offset int) ([]*entity.Company, error)
	ListForAdmin(limit, offset int) ([]*entity.Company, error)
	Delete(id string) error

	// HasActiveModule informa si la empresa tiene el módulo activo y no vencido.
	HasActiveModule(ctx context.Context, companyID, moduleName string) (bool, error)

	// ListModules devuelve los módulos contratados por la empresa.
	ListModules(ctx context.Context, companyID string) ([]*entity.CompanyModule, error)

	// GetModule devuelve un módulo específico de la empresa.
	GetModule(ctx context.Context, companyID, moduleName string) (*entity.CompanyModule, error)

	// UpsertModule crea o actualiza el módulo de la empresa.
	UpsertModule(ctx context.Context, module *entity.CompanyModule) error

	// DeleteModule elimina un módulo asignado a la empresa.
	DeleteModule(ctx context.Context, companyID, moduleName string) error

	// HasActiveScreen informa si la empresa tiene la pantalla activa.
	HasActiveScreen(ctx context.Context, companyID, screenID string) (bool, error)

	// ListScreens devuelve las pantallas habilitadas para la empresa.
	ListScreens(ctx context.Context, companyID string) ([]*entity.CompanyScreen, error)

	// GetScreen devuelve el estado de una pantalla para una empresa.
	GetScreen(ctx context.Context, companyID, screenID string) (*entity.CompanyScreen, error)

	// UpsertScreen crea o actualiza una pantalla habilitada para una empresa.
	UpsertScreen(ctx context.Context, screen *entity.CompanyScreen) error

	// ReplaceScreens reemplaza todas las pantallas habilitadas de una empresa.
	ReplaceScreens(ctx context.Context, companyID string, screenIDs []string) error

	// DeleteScreen desactiva o elimina una pantalla para una empresa.
	DeleteScreen(ctx context.Context, companyID, screenID string) error
}
