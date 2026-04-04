package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// AdminUserUseCase define el contrato para administrar usuarios de una empresa desde super_admin.
type AdminUserUseCase interface {
	ListByCompany(companyID string, limit, offset int) ([]*dto.UserResponse, error)
	CreateForCompany(companyID string, in dto.AdminCreateUserRequest) (*dto.UserResponse, error)
	UpdateForCompany(companyID, userID string, in dto.AdminUpdateUserRequest) (*dto.UserResponse, error)
}

// AdminUserHandler expone endpoints de administración de usuarios por empresa.
type AdminUserHandler struct {
	uc AdminUserUseCase
}

// NewAdminUserHandler construye el handler.
func NewAdminUserHandler(uc AdminUserUseCase) *AdminUserHandler {
	return &AdminUserHandler{uc: uc}
}

// ListByCompany lista usuarios de una empresa.
func (h *AdminUserHandler) ListByCompany(c *fiber.Ctx) error {
	companyID := c.Params("company_id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "company_id es requerido"})
	}
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)
	list, err := h.uc.ListByCompany(companyID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(list)
}

// CreateForCompany crea un usuario admin para una empresa.
func (h *AdminUserHandler) CreateForCompany(c *fiber.Ctx) error {
	companyID := c.Params("company_id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_ID", Message: "company_id es requerido"})
	}
	var in dto.AdminCreateUserRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.uc.CreateForCompany(companyID, in)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		case domain.ErrEmailAlreadyExists:
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "EMAIL_EXISTS", Message: "el email ya está registrado en esta empresa"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// UpdateForCompany actualiza un usuario de la empresa.
func (h *AdminUserHandler) UpdateForCompany(c *fiber.Ctx) error {
	companyID := c.Params("company_id")
	userID := c.Params("user_id")
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "MISSING_PARAMS", Message: "company_id y user_id son requeridos"})
	}
	var in dto.AdminUpdateUserRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.uc.UpdateForCompany(companyID, userID, in)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "usuario no encontrado"})
		case domain.ErrForbidden:
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		case domain.ErrEmailAlreadyExists:
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "EMAIL_EXISTS", Message: "el email ya está registrado en esta empresa"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}
