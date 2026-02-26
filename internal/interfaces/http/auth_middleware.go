package http

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/pkg/jwt"
)

// ── Claves de contexto (c.Locals) ────────────────────────────────────────────

const (
	LocalUserID    = "user_id"
	LocalCompanyID = "company_id"
	LocalRole      = "role" // "admin" | "bodeguero" | "vendedor"
)

// ── AuthMiddleware ─────────────────────────────────────────────────────────────

// AuthMiddleware valida el Bearer Token JWT y almacena userID, companyID y role
// en c.Locals para que los handlers y middlewares posteriores los consuman.
func AuthMiddleware(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Code: "MISSING_TOKEN", Message: "Authorization header requerido",
			})
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Code: "INVALID_TOKEN", Message: "formato: Bearer <token>",
			})
		}
		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Code: "MISSING_TOKEN", Message: "token vacío",
			})
		}

		userID, companyID, role, err := jwt.Parse(jwtSecret, tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Code: "INVALID_TOKEN", Message: "token inválido o expirado",
			})
		}

		c.Locals(LocalUserID, userID)
		c.Locals(LocalCompanyID, companyID)
		c.Locals(LocalRole, role)
		return c.Next()
	}
}

// ── RBAC Middleware ────────────────────────────────────────────────────────────

// RequireRole devuelve un middleware que permite el acceso solo si el rol del
// usuario (extraído del JWT por AuthMiddleware) está dentro de allowedRoles.
//
// Uso en el router:
//
//	group.Post("/", RequireRole(entity.RoleAdmin, entity.RoleVendedor), handler)
//
// Si el rol no está permitido devuelve 403 Forbidden con un mensaje descriptivo.
// Si el rol no está en el contexto (token sin rol, token antiguo) devuelve 401.
func RequireRole(allowedRoles ...string) fiber.Handler {
	// Construir un set para búsqueda O(1)
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		role := GetRole(c)
		if role == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Code:    "MISSING_ROLE",
				Message: "el token no contiene información de rol; vuelve a iniciar sesión",
			})
		}
		if _, ok := allowed[role]; !ok {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
				Code:    "FORBIDDEN",
				Message: fmt.Sprintf("acceso denegado: se requiere rol %s", strings.Join(allowedRoles, " o ")),
			})
		}
		return c.Next()
	}
}

// ── Helpers de contexto ───────────────────────────────────────────────────────

// GetUserID devuelve el UserID almacenado por AuthMiddleware.
func GetUserID(c *fiber.Ctx) string {
	v, _ := c.Locals(LocalUserID).(string)
	return v
}

// GetCompanyID devuelve el CompanyID almacenado por AuthMiddleware.
func GetCompanyID(c *fiber.Ctx) string {
	v, _ := c.Locals(LocalCompanyID).(string)
	return v
}

// GetRole devuelve el Role almacenado por AuthMiddleware.
// Devuelve "" si el token no incluía claim de rol (tokens emitidos antes de este cambio).
func GetRole(c *fiber.Ctx) string {
	v, _ := c.Locals(LocalRole).(string)
	return v
}

// IsAdmin es un helper semántico para comprobar si el usuario tiene rol admin.
func IsAdmin(c *fiber.Ctx) bool {
	return GetRole(c) == entity.RoleAdmin
}
