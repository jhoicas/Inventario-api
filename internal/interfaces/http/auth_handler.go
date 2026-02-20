package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/tu-usuario/inventory-pro/internal/application/auth"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/domain"
)

// AuthHandler maneja registro y login.
type AuthHandler struct {
	uc *auth.AuthUseCase
}

// NewAuthHandler construye el handler de auth.
func NewAuthHandler(uc *auth.AuthUseCase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

// Register godoc
// @Summary      Registrar usuario
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  dto.RegisterRequest  true  "email, password, company_id"
// @Success      201   {object}  dto.UserResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Router       /api/auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var in dto.RegisterRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	if in.Email == "" || in.Password == "" || in.CompanyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "email, password y company_id son requeridos"})
	}
	if len(in.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "password debe tener al menos 8 caracteres"})
	}
	user, err := h.uc.RegisterUser(in)
	if err != nil {
		if err == domain.ErrEmailAlreadyExists {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "EMAIL_EXISTS", Message: "el email ya está registrado en esta empresa"})
		}
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "COMPANY_NOT_FOUND", Message: "la empresa no existe"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(user)
}

// Login godoc
// @Summary      Iniciar sesión
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  dto.LoginRequest  true  "email, password"
// @Success      200   {object}  dto.LoginResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Router       /api/auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var in dto.LoginRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	if in.Email == "" || in.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "email y password son requeridos"})
	}
	out, err := h.uc.Login(in)
	if err != nil {
		if err == domain.ErrUserNotFound || err == domain.ErrUnauthorized {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "credenciales inválidas"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "cuenta inactiva o suspendida"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}
