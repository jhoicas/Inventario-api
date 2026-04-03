package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
)

type moduleAccessChecker interface {
	HasActiveModule(ctx context.Context, companyID, moduleName string) (bool, error)
}

type screenAccessChecker interface {
	CanAccess(ctx context.Context, roleRef, apiEndpoint string) (bool, error)
}

// RequireAccess valida en un solo middleware:
// 1) módulo activo para la empresa (si moduleName != "")
// 2) permiso de pantalla por rol (role_screens + screens)
//
// Debe usarse después de AuthMiddleware.
func RequireAccess(moduleName string, moduleChecker moduleAccessChecker, screenChecker screenAccessChecker) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := GetCompanyID(c)
		if companyID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Code:    "UNAUTHORIZED",
				Message: "company_id no encontrado en el token",
			})
		}

		// Admin/super-admin bypass en pantalla y módulo.
		if IsAdmin(c) || IsSuperAdmin(c) {
			return c.Next()
		}

		if moduleName != "" && moduleChecker != nil {
			active, err := moduleChecker.HasActiveModule(c.Context(), companyID, moduleName)
			if err != nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{
					Code:    "MODULE_CHECK_FAILED",
					Message: "no se pudo verificar el módulo, intente más tarde",
				})
			}
			if !active {
				return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
					Code:    "MODULE_DISABLED",
					Message: "el módulo '" + moduleName + "' no está activo para esta empresa",
				})
			}
		}

		if screenChecker != nil {
			roleRef := GetRoleRef(c)
			if roleRef == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
					Code:    "MISSING_ROLE",
					Message: "el token no contiene rol activo",
				})
			}
			endpoint := normalizeRBACEndpoint(c.Path())
			allowed, err := screenChecker.CanAccess(c.Context(), roleRef, endpoint)
			if err != nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{
					Code:    "RBAC_CHECK_FAILED",
					Message: "no se pudo verificar el permiso de la ruta",
				})
			}
			if !allowed {
				return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
					Code:    "FORBIDDEN",
					Message: "acceso denegado a la ruta " + endpoint,
				})
			}
		}

		return c.Next()
	}
}
