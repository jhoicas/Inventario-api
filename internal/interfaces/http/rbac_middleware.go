package http

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// rbacAccessChecker es el contrato mínimo para validar rutas dinámicas.
type rbacAccessChecker interface {
	CanAccess(ctx context.Context, roleRef, apiEndpoint string) (bool, error)
	GetScreenByEndpoint(ctx context.Context, apiEndpoint string) (*entity.Screen, error)
}

type companyScreenAccessChecker interface {
	HasActiveScreen(ctx context.Context, companyID, screenID string) (bool, error)
}

// RequirePermission valida que el rol activo pueda acceder a la ruta actual.
// Debe usarse después de AuthMiddleware y de RequireModule en los módulos SaaS.
func RequirePermission(checker rbacAccessChecker, companyChecker companyScreenAccessChecker) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Regla: admin/super_admin siempre pasan (evita bloqueos por desincronización de catálogo RBAC en DB).
		if IsAdmin(c) || IsSuperAdmin(c) {
			return c.Next()
		}

		companyID := GetCompanyID(c)
		if companyID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Code:    "UNAUTHORIZED",
				Message: "company_id no encontrado en el token",
			})
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

		if companyChecker != nil {
			screen, err := checker.GetScreenByEndpoint(c.Context(), endpoint)
			if err != nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{
					Code:    "SCREEN_LOOKUP_FAILED",
					Message: "no se pudo verificar la pantalla para la ruta",
				})
			}
			if screen == nil || screen.ID == "" {
				return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
					Code:    "SCREEN_NOT_REGISTERED",
					Message: "la ruta no está registrada en RBAC",
				})
			}
			active, err := companyChecker.HasActiveScreen(c.Context(), companyID, screen.ID)
			if err != nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{
					Code:    "COMPANY_SCREEN_CHECK_FAILED",
					Message: "no se pudo verificar el acceso de la empresa a la ruta",
				})
			}
			if !active {
				return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
					Code:    "COMPANY_SCREEN_DISABLED",
					Message: "acceso denegado a la ruta para esta empresa",
				})
			}
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
