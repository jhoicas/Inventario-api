package http

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
)

// rbacAccessChecker es el contrato mínimo para validar rutas dinámicas.
type rbacAccessChecker interface {
	CanAccess(ctx context.Context, roleRef, apiEndpoint string) (bool, error)
}

// RequirePermission valida que el rol activo pueda acceder a la ruta actual.
// Debe usarse después de AuthMiddleware y de RequireModule en los módulos SaaS.
func RequirePermission(checker rbacAccessChecker) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Regla: admin siempre pasa (evita bloqueos por desincronización de catálogo RBAC en DB).
		if IsAdmin(c) {
			return c.Next()
		}

		roleRef := GetRoleRef(c)
		if roleRef == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Code:    "MISSING_ROLE",
				Message: "el token no contiene rol activo",
			})
		}

		endpoint := currentEndpoint(c)
		allowed, err := checker.CanAccess(c.Context(), roleRef, endpoint)
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
		return c.Next()
	}
}

func currentEndpoint(c *fiber.Ctx) string {
	return normalizeRBACEndpoint(c.Path())
}

func normalizeRBACEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return endpoint
	}
	switch endpoint {
	case "/api/dian/settings", "/api/dian/configuration":
		endpoint = "/api/settings/dian"
	}
	if strings.HasSuffix(endpoint, "/") && endpoint != "/" {
		endpoint = strings.TrimRight(endpoint, "/")
	}
	return endpoint
}

