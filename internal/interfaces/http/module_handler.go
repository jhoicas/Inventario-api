package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
)

// ModuleCatalogUseCase define el contrato para listar módulos desde super admin.
type ModuleCatalogUseCase interface {
	ListCatalog(ctx context.Context) (*dto.RBACCatalogResponse, error)
}

// ModuleHandler expone endpoints de módulos para administración.
type ModuleHandler struct {
	uc ModuleCatalogUseCase
}

// NewModuleHandler construye el handler de módulos.
func NewModuleHandler(uc ModuleCatalogUseCase) *ModuleHandler {
	return &ModuleHandler{uc: uc}
}

// ListModules devuelve el catálogo de módulos y pantallas.
func (h *ModuleHandler) ListModules(c *fiber.Ctx) error {
	out, err := h.uc.ListCatalog(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(out)
}
