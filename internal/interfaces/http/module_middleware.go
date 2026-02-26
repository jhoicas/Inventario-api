package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
)

// moduleChecker es el contrato mínimo que necesita el middleware para verificar módulos.
// Lo implementa *usecase.ModuleService; el uso de interfaz evita el import circular.
type moduleChecker interface {
	HasActiveModule(ctx context.Context, companyID, moduleName string) (bool, error)
}

// RequireModule devuelve un middleware Fiber que verifica si la empresa del token JWT
// tiene el módulo activo. Debe usarse DESPUÉS de AuthMiddleware (necesita LocalCompanyID).
//
// Comportamiento:
//   - 403 Forbidden  → módulo no contratado o vencido.
//   - 503 Service Unavailable → fallo de infraestructura al consultar la DB.
//   - Si no hay company_id en el contexto, responde 401 (el AuthMiddleware debería haberlo puesto).
func RequireModule(moduleName string, checker moduleChecker) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID := GetCompanyID(c)
		if companyID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Code:    "UNAUTHORIZED",
				Message: "company_id no encontrado en el token",
			})
		}

		active, err := checker.HasActiveModule(c.Context(), companyID, moduleName)
		if err != nil {
			// Fallo de infraestructura: no bloquear al usuario por un error de DB,
			// pero sí registrar. Aquí se puede inyectar un logger si se necesita.
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

		return c.Next()
	}
}
