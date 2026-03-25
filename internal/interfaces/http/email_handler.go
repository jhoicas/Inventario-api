package http

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

type EmailUseCase interface {
	CreateAccount(companyID string, in dto.CreateEmailAccountRequest) (*dto.EmailAccountResponse, error)
	UpdateAccount(companyID, id string, in dto.UpdateEmailAccountRequest) (*dto.EmailAccountResponse, error)
	DeleteAccount(companyID, id string) error
	GetAccount(companyID, id string) (*dto.EmailAccountResponse, error)
	ListAccounts(companyID string, limit, offset int) ([]dto.EmailAccountResponse, error)
	TestConnection(companyID, id string) (*dto.TestIMAPConnectionResponse, error)
	ListEmails(companyID, customerID string, isRead *bool, limit, offset int) (*dto.EmailListResponse, error)
	GetEmailAndMarkAsRead(companyID, id string) (*dto.EmailResponse, error)
	CreateTicketFromEmail(companyID, userID, emailID string) (*dto.CreateTicketFromEmailResponse, error)
}

type EmailHandler struct {
	uc EmailUseCase
}

func NewEmailHandler(uc EmailUseCase) *EmailHandler {
	return &EmailHandler{uc: uc}
}

func (h *EmailHandler) ListEmailAccounts(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	items, err := h.uc.ListAccounts(companyID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(items)
}

func (h *EmailHandler) CreateEmailAccount(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.CreateEmailAccountRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.uc.CreateAccount(companyID, in)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "email_address, imap_server, imap_port y password son requeridos"})
		case domain.ErrDuplicate:
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "la cuenta de correo ya existe para la compañía"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
		}
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *EmailHandler) GetEmailAccount(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	id := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	out, err := h.uc.GetAccount(companyID, id)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id inválido"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "cuenta no encontrada"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
		}
	}
	return c.JSON(out)
}

func (h *EmailHandler) UpdateEmailAccount(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	id := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.UpdateEmailAccountRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.uc.UpdateAccount(companyID, id, in)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "datos inválidos"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "cuenta no encontrada"})
		case domain.ErrDuplicate:
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "DUPLICATE", Message: "la cuenta de correo ya existe para la compañía"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
		}
	}
	return c.JSON(out)
}

func (h *EmailHandler) DeleteEmailAccount(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	id := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if err := h.uc.DeleteAccount(companyID, id); err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id inválido"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "cuenta no encontrada"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *EmailHandler) TestEmailAccountConnection(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	id := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	out, err := h.uc.TestConnection(companyID, id)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id inválido"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "cuenta no encontrada"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
		}
	}
	if !out.Success {
		return c.Status(fiber.StatusBadGateway).JSON(out)
	}
	return c.JSON(out)
}

func (h *EmailHandler) ListEmails(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	customerID := c.Query("customer_id")
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	var isRead *bool
	if isReadQ := c.Query("is_read"); isReadQ != "" {
		parsed, err := strconv.ParseBool(isReadQ)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "is_read debe ser true o false"})
		}
		isRead = &parsed
	}

	out, err := h.uc.ListEmails(companyID, customerID, isRead, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

func (h *EmailHandler) GetEmail(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	id := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	out, err := h.uc.GetEmailAndMarkAsRead(companyID, id)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id inválido"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "correo no encontrado"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
		}
	}
	return c.JSON(out)
}

func (h *EmailHandler) CreateTicketFromEmail(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	emailID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	out, err := h.uc.CreateTicketFromEmail(companyID, userID, emailID)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id inválido"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "correo no encontrado"})
		case domain.ErrConflict:
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{Code: "CONFLICT", Message: "el correo no está asociado a un cliente"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
		}
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}
