package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/pkg/jwt"
)

// Locals keys para UserID y CompanyID en Fiber.
const (
	LocalUserID    = "user_id"
	LocalCompanyID = "company_id"
)

// AuthMiddleware valida el Bearer Token JWT y extrae UserID y CompanyID a c.Locals.
func AuthMiddleware(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "Authorization header requerido"})
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "INVALID_TOKEN", Message: "formato: Bearer <token>"})
		}
		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "token vacío"})
		}
		userID, companyID, err := jwt.Parse(jwtSecret, tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "INVALID_TOKEN", Message: "token inválido o expirado"})
		}
		c.Locals(LocalUserID, userID)
		c.Locals(LocalCompanyID, companyID)
		return c.Next()
	}
}

// GetUserID devuelve el UserID del contexto (después del middleware de auth).
func GetUserID(c *fiber.Ctx) string {
	v := c.Locals(LocalUserID)
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// GetCompanyID devuelve el CompanyID del contexto (después del middleware de auth).
func GetCompanyID(c *fiber.Ctx) string {
	v := c.Locals(LocalCompanyID)
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
