package usecase

import (
	"context"
	"fmt"

	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
)

// ModuleService verifica qué módulos SaaS tiene activos una empresa.
// Es el único punto de la aplicación que conoce la lógica de activación de módulos.
type ModuleService struct {
	companyRepo repository.CompanyRepository
}

// NewModuleService construye el servicio de módulos.
func NewModuleService(companyRepo repository.CompanyRepository) *ModuleService {
	return &ModuleService{companyRepo: companyRepo}
}

// HasActiveModule informa si la empresa tiene el módulo activo y sin vencer.
// Devuelve false (sin error) si la empresa no tiene el módulo contratado.
// Devuelve error solo ante fallos de infraestructura (DB caída, timeout, etc.).
func (s *ModuleService) HasActiveModule(ctx context.Context, companyID, moduleName string) (bool, error) {
	if companyID == "" || moduleName == "" {
		return false, fmt.Errorf("module: companyID y moduleName son obligatorios")
	}
	return s.companyRepo.HasActiveModule(ctx, companyID, moduleName)
}
