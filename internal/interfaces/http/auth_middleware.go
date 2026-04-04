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
	LocalRoleID    = "role_id"
	LocalRole      = "role"  // compatibilidad: primer rol
	LocalRoles     = "roles" // slice completo de roles
)

// ── AuthMiddleware ─────────────────────────────────────────────────────────────

// AuthMiddleware valida el Bearer Token JWT y almacena userID, companyID y roles
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

		claims, err := jwt.ParseClaims(jwtSecret, tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Code: "INVALID_TOKEN", Message: "token inválido o expirado",
			})
		}

		c.Locals(LocalUserID, claims.UserID)
		c.Locals(LocalCompanyID, claims.CompanyID)
		c.Locals(LocalRoles, claims.Roles)
		c.Locals(LocalRoleID, claims.RoleID)
		if len(claims.Roles) > 0 {
			// Para compatibilidad con código que solo lee un rol:
			c.Locals(LocalRole, claims.Roles[0])
		}
		return c.Next()
	}
}

// ── RBAC Middleware ────────────────────────────────────────────────────────────

// RequireRole (RBAC) permite acceso si el usuario:
//   - tiene rol "admin" en sus claims, O
//   - tiene al menos uno de los roles pasados en allowedRoles.
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
		roles := GetRoles(c)
		if len(roles) == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{
				Code:    "MISSING_ROLE",
				Message: "el token no contiene información de roles; vuelve a iniciar sesión",
			})
		}
		// Acceso siempre permitido para admin/super_admin.
		for _, r := range roles {
			if r == entity.RoleAdmin || r == entity.RoleSuperAdmin {
				return c.Next()
			}
		}
		// Si tiene al menos uno de los roles requeridos se permite acceso.
		for _, r := range roles {
			if _, ok := allowed[r]; ok {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{
			Code:    "FORBIDDEN",
			Message: fmt.Sprintf("acceso denegado: se requiere rol %s", strings.Join(allowedRoles, " o ")),
		})
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

// GetRoles devuelve la lista completa de roles almacenada por AuthMiddleware.
func GetRoles(c *fiber.Ctx) []string {
	if v, ok := c.Locals(LocalRoles).([]string); ok && len(v) > 0 {
		return v
	}
	// Compatibilidad: tokens antiguos que solo tenían un "role" string.
	if s, ok := c.Locals(LocalRole).(string); ok && s != "" {
		return []string{s}
	}
	return nil
}

// GetRole devuelve el primer rol almacenado por AuthMiddleware (helper legacy).
// Devuelve "" si el token no incluía claims de rol.
func GetRole(c *fiber.Ctx) string {
	roles := GetRoles(c)
	if len(roles) == 0 {
		return ""
	}
	return roles[0]
}

// GetRoleID devuelve el role_id almacenado por AuthMiddleware si existe.
func GetRoleID(c *fiber.Ctx) string {
	v, _ := c.Locals(LocalRoleID).(string)
	return v
}

// GetRoleRef devuelve el identificador más útil para RBAC: role_id si existe, si no el primer role.
func GetRoleRef(c *fiber.Ctx) string {
	if roleID := GetRoleID(c); roleID != "" {
		return roleID
	}
	return GetRole(c)
}

// IsAdmin es un helper semántico para comprobar si el usuario tiene rol admin.
func IsAdmin(c *fiber.Ctx) bool {
	roles := GetRoles(c)
	for _, r := range roles {
		if r == entity.RoleAdmin {
			return true
		}
	}
	return false
}

// IsSuperAdmin comprueba si el usuario tiene el rol super_admin.
func IsSuperAdmin(c *fiber.Ctx) bool {
	roles := GetRoles(c)
	for _, r := range roles {
		if r == entity.RoleSuperAdmin {
			return true
		}
	}
	return false
}
