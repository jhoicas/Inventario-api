package http

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"golang.org/x/crypto/bcrypt"
)

// UserHandler maneja las peticiones HTTP de gestión de usuarios (solo admin).
type UserHandler struct {
	repo repository.UserRepository
}

// NewUserHandler construye el handler de usuarios.
func NewUserHandler(repo repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

// List godoc
// @Summary      Listar usuarios
// @Description  Lista los usuarios de la empresa autenticada (solo admin)
// @Tags         users
// @Security     Bearer
// @Produce      json
// @Param        limit   query     int   false  "Límite"   default(20)
// @Param        offset  query     int   false  "Offset"   default(0)
// @Success      200     {array}   dto.UserResponse
// @Failure      401     {object}  dto.ErrorResponse
// @Failure      500     {object}  dto.ErrorResponse
// @Router       /api/users [get]
func (h *UserHandler) List(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	list, err := h.repo.ListByCompany(companyID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	out := make([]dto.UserResponse, 0, len(list))
	for _, u := range list {
		out = append(out, dto.UserResponse{
			ID:        u.ID,
			CompanyID: u.CompanyID,
			Email:     u.Email,
			Name:      u.Name,
			Roles:     u.Roles,
			Status:    u.Status,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		})
	}
	return c.JSON(out)
}

// Create godoc
// @Summary      Crear usuario
// @Description  Crea un usuario dentro de la empresa autenticada (solo admin)
// @Tags         users
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateUserRequest  true  "Datos del usuario (password plano, se hashea)"
// @Success      201   {object}  dto.UserResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /api/users [post]
func (h *UserHandler) Create(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.CreateUserRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	if in.Email == "" || in.Password == "" || in.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "email, password y name son requeridos"})
	}
	if len(in.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "password debe tener al menos 8 caracteres"})
	}
	roles := in.Roles
	if len(roles) == 0 {
		roles = []string{entity.RoleSales}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	now := time.Now()
	user := &entity.User{
		ID:           uuid.New().String(),
		CompanyID:    companyID,
		Email:        in.Email,
		PasswordHash: string(hash),
		Name:         in.Name,
		Roles:        roles,
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.repo.Create(user); err != nil {
		if err == domain.ErrEmailAlreadyExists {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "EMAIL_EXISTS", Message: "el email ya está registrado en esta empresa"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(dto.UserResponse{
		ID:        user.ID,
		CompanyID: user.CompanyID,
		Email:     user.Email,
		Name:      user.Name,
		Roles:     user.Roles,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

// Update godoc
// @Summary      Actualizar usuario
// @Description  Actualiza roles o estado de un usuario (solo admin)
// @Tags         users
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "ID del usuario"
// @Param        body  body      object  true  "{\"roles\": [\"admin\"], \"status\": \"active\"}"
// @Success      200   {object}  dto.UserResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      401   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /api/users/{id} [put]
func (h *UserHandler) Update(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	id := c.Params("id")
	if companyID == "" || id == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	user, err := h.repo.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	if user == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "usuario no encontrado"})
	}
	if user.CompanyID != companyID {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
	}
	var body struct {
		Roles  []string `json:"roles"`
		Status string   `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	if len(body.Roles) > 0 {
		user.Roles = body.Roles
	}
	if body.Status != "" {
		user.Status = body.Status
	}
	user.UpdatedAt = time.Now()
	if err := h.repo.Update(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(dto.UserResponse{
		ID:        user.ID,
		CompanyID: user.CompanyID,
		Email:     user.Email,
		Name:      user.Name,
		Roles:     user.Roles,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

