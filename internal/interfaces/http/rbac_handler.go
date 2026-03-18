package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// RBACUseCase define lo que necesita el handler de administración RBAC.
type RBACUseCase interface {
	ListRoles(ctx context.Context) ([]dto.RoleResponse, error)
	ListCatalog(ctx context.Context) (*dto.RBACCatalogResponse, error)
	GetMenuByRoleRef(ctx context.Context, roleRef string) (*dto.RBACMenuResponse, error)
	UpdateRoleScreens(ctx context.Context, roleRef string, in dto.UpdateRoleScreensRequest) (*dto.RBACMenuResponse, error)
}

// RBACHandler expone endpoints para menú y permisos.
type RBACHandler struct {
	uc RBACUseCase
}

// NewRBACHandler construye el handler RBAC.
func NewRBACHandler(uc RBACUseCase) *RBACHandler {
	return &RBACHandler{uc: uc}
}

// ListRoles godoc
// @Summary      Listar roles
// @Tags         rbac
// @Security     Bearer
// @Produce      json
// @Success      200  {array}   dto.RoleResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/rbac/roles [get]
func (h *RBACHandler) ListRoles(c *fiber.Ctx) error {
	out, err := h.uc.ListRoles(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// GetCatalog godoc
// @Summary      Catálogo de módulos y pantallas
// @Tags         rbac
// @Security     Bearer
// @Produce      json
// @Success      200  {object}  dto.RBACCatalogResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/rbac/modules [get]
func (h *RBACHandler) GetCatalog(c *fiber.Ctx) error {
	out, err := h.uc.ListCatalog(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// GetCurrentMenu godoc
// @Summary      Menú del rol activo
// @Tags         rbac
// @Security     Bearer
// @Produce      json
// @Success      200  {object}  dto.RBACMenuResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/rbac/menu [get]
func (h *RBACHandler) GetCurrentMenu(c *fiber.Ctx) error {
	roleRef := GetRoleRef(c)
	if roleRef == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "MISSING_ROLE", Message: "el token no contiene rol activo"})
	}
	out, err := h.uc.GetMenuByRoleRef(c.Context(), roleRef)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "role_id requerido"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "rol no encontrado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// GetRoleMenu godoc
// @Summary      Menú por role_id
// @Tags         rbac
// @Security     Bearer
// @Produce      json
// @Param        role_id  path  string  true  "Role ID o key"
// @Success      200      {object}  dto.RBACMenuResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      401      {object}  dto.ErrorResponse
// @Failure      404      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /api/rbac/roles/{role_id}/menu [get]
func (h *RBACHandler) GetRoleMenu(c *fiber.Ctx) error {
	roleRef := c.Params("role_id")
	if roleRef == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "role_id requerido"})
	}
	out, err := h.uc.GetMenuByRoleRef(c.Context(), roleRef)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "role_id inválido"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "rol no encontrado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// UpdateRoleScreens godoc
// @Summary      Actualizar pantallas de un rol
// @Tags         rbac
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        role_id  path  string  true  "Role ID o key"
// @Param        body      body  dto.UpdateRoleScreensRequest  true  "screen_ids"
// @Success      200       {object}  dto.RBACMenuResponse
// @Failure      400       {object}  dto.ErrorResponse
// @Failure      401       {object}  dto.ErrorResponse
// @Failure      404       {object}  dto.ErrorResponse
// @Failure      500       {object}  dto.ErrorResponse
// @Router       /api/rbac/roles/{role_id}/screens [put]
func (h *RBACHandler) UpdateRoleScreens(c *fiber.Ctx) error {
	roleRef := c.Params("role_id")
	if roleRef == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "role_id requerido"})
	}
	var in dto.UpdateRoleScreensRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.uc.UpdateRoleScreens(c.Context(), roleRef, in)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "role_id inválido"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "rol no encontrado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

