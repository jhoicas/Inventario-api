package http

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/crm"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// CRMHandler maneja las peticiones HTTP del módulo CRM (protegido + RequireModule crm).
type CRMHandler struct {
	LoyaltyUC *crm.LoyaltyUseCase
	TaskUC    *crm.TaskUseCase
	PQRUC     *crm.PQRUseCase
	AICRMUC   *crm.AICRMUseCase
	InteractionRepo interface {
		Create(interaction *entity.CRMInteraction) error
		ListByCustomer(customerID string, limit, offset int) ([]*entity.CRMInteraction, error)
	}
}

// NewCRMHandler construye el handler.
func NewCRMHandler(
	loyaltyUC *crm.LoyaltyUseCase,
	taskUC *crm.TaskUseCase,
	pqrUC *crm.PQRUseCase,
	aiCRMUC *crm.AICRMUseCase,
	interactionRepo interface {
		Create(interaction *entity.CRMInteraction) error
		ListByCustomer(customerID string, limit, offset int) ([]*entity.CRMInteraction, error)
	},
) *CRMHandler {
	return &CRMHandler{
		LoyaltyUC:       loyaltyUC,
		TaskUC:          taskUC,
		PQRUC:           pqrUC,
		AICRMUC:         aiCRMUC,
		InteractionRepo: interactionRepo,
	}
}

// GetProfile360 obtiene la vista 360 del cliente.
// @Summary      Vista 360 del cliente
// @Description  Obtiene la vista 360 del cliente con datos base, perfil CRM y categoría de fidelización
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Customer ID"
// @Success      200  {object}  dto.Profile360Response
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      403  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /api/crm/customers/{id}/profile360 [get]
func (h *CRMHandler) GetProfile360(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	customerID := c.Params("id")
	if companyID == "" || customerID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	out, err := h.LoyaltyUC.GetProfile360(c.Context(), companyID, customerID)
	if err != nil {
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "cliente no encontrado"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// AssignCategory asigna categoría de fidelización al cliente.
// @Summary      Asignar categoría al cliente
// @Description  Asigna o actualiza la categoría de fidelización y el LTV de un cliente
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Customer ID"
// @Param        body body      dto.AssignCategoryRequest true "Category and LTV"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /api/crm/customers/{id}/category [put]
func (h *CRMHandler) AssignCategory(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	customerID := c.Params("id")
	if companyID == "" || customerID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.AssignCategoryRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	err := h.LoyaltyUC.AssignCategory(c.Context(), companyID, customerID, in)
	if err != nil {
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "cliente o categoría no encontrado"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// ListCategories lista categorías de fidelización.
// @Summary      Listar categorías CRM
// @Description  Lista las categorías de fidelización configuradas para la empresa
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        limit  query     int  false  "Limit"
// @Param        offset query     int  false  "Offset"
// @Success      200    {array}   dto.CategoryResponse
// @Router       /api/crm/categories [get]
func (h *CRMHandler) ListCategories(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	list, err := h.LoyaltyUC.ListCategories(c.Context(), companyID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(list)
}

// ListBenefitsByCategory lista beneficios de una categoría.
// @Summary      Listar beneficios por categoría
// @Description  Lista los beneficios asociados a una categoría de fidelización
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        id     path      string  true  "Category ID"
// @Param        limit  query     int     false "Limit"
// @Param        offset query     int     false "Offset"
// @Success      200    {array}   dto.BenefitResponse
// @Router       /api/crm/categories/{id}/benefits [get]
func (h *CRMHandler) ListBenefitsByCategory(c *fiber.Ctx) error {
	categoryID := c.Params("id")
	if categoryID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "id requerido"})
	}
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	list, err := h.LoyaltyUC.ListBenefitsByCategory(c.Context(), categoryID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(list)
}

// CreateTask crea una tarea.
// @Summary      Crear tarea CRM
// @Description  Crea una tarea de seguimiento o gestión comercial para un cliente
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateTaskRequest  true  "Task"
// @Success      201   {object}  dto.TaskResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Router       /api/crm/tasks [post]
func (h *CRMHandler) CreateTask(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.CreateTaskRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.TaskUC.Create(c.Context(), companyID, userID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "titulo requerido"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// GetTask obtiene una tarea por ID.
// @Summary      Obtener tarea
// @Description  Obtiene el detalle de una tarea CRM por su identificador
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Task ID"
// @Success      200  {object}  dto.TaskResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /api/crm/tasks/{id} [get]
func (h *CRMHandler) GetTask(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	id := c.Params("id")
	if companyID == "" || id == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	out, err := h.TaskUC.GetByID(c.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "tarea no encontrada"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// UpdateTask actualiza una tarea.
// @Summary      Actualizar tarea
// @Description  Actualiza los datos y el estado de una tarea CRM
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "Task ID"
// @Param        body  body      dto.UpdateTaskRequest  true  "Updates"
// @Success      200   {object}  dto.TaskResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Router       /api/crm/tasks/{id} [put]
func (h *CRMHandler) UpdateTask(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	id := c.Params("id")
	if companyID == "" || id == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.UpdateTaskRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.TaskUC.Update(c.Context(), companyID, id, in)
	if err != nil {
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "tarea no encontrada"})
		}
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "status inválido"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// ListTasks lista tareas de la empresa.
// @Summary      Listar tareas
// @Description  Lista las tareas CRM de la empresa, opcionalmente filtradas por estado
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        limit  query     int    false "Limit"
// @Param        offset query     int    false "Offset"
// @Param        status query     string false "Filter by status"
// @Success      200    {object}  dto.TaskResponseList
// @Router       /api/crm/tasks [get]
func (h *CRMHandler) ListTasks(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	status := c.Query("status")
	out, err := h.TaskUC.ListByCompany(c.Context(), companyID, status, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// CreateInteraction registra una interacción.
// @Summary      Registrar interacción
// @Description  Registra una interacción con el cliente (llamada, correo, reunión, etc.)
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateInteractionRequest  true  "Interaction"
// @Success      201   {object}  dto.InteractionResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Router       /api/crm/interactions [post]
func (h *CRMHandler) CreateInteraction(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.CreateInteractionRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	if in.CustomerID == "" || in.Type == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "customer_id y type requeridos"})
	}
	typ := entity.InteractionType(in.Type)
	if typ != entity.InteractionTypeCall && typ != entity.InteractionTypeEmail && typ != entity.InteractionTypeMeeting && typ != entity.InteractionTypeOther {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "type debe ser call, email, meeting u other"})
	}
	now := time.Now()
	m := &entity.CRMInteraction{
		ID:         uuid.New().String(),
		CompanyID:  companyID,
		CustomerID: in.CustomerID,
		Type:       typ,
		Subject:    in.Subject,
		Body:       in.Body,
		CreatedBy:  userID,
		CreatedAt:  now,
	}
	if err := h.InteractionRepo.Create(m); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	resp := dto.InteractionResponse{
		ID:         m.ID,
		CompanyID:  m.CompanyID,
		CustomerID: m.CustomerID,
		Type:       in.Type,
		Subject:    m.Subject,
		Body:       m.Body,
		CreatedBy:  m.CreatedBy,
		CreatedAt:  m.CreatedAt,
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// CreateTicket radica un ticket PQR.
// @Summary      Radicar ticket PQR
// @Description  Radica un nuevo caso PQR asociado a un cliente y analiza su sentimiento
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateTicketRequest  true  "Ticket"
// @Success      201   {object}  dto.TicketResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Router       /api/crm/tickets [post]
func (h *CRMHandler) CreateTicket(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.CreateTicketRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.PQRUC.Create(c.Context(), companyID, userID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "customer_id, subject y description requeridos"})
		}
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "cliente no encontrado"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// GetTicket obtiene un ticket por ID.
// @Summary      Obtener ticket PQR
// @Description  Obtiene el detalle de un ticket PQR por su identificador
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Ticket ID"
// @Success      200  {object}  dto.TicketResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /api/crm/tickets/{id} [get]
func (h *CRMHandler) GetTicket(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	id := c.Params("id")
	if companyID == "" || id == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	out, err := h.PQRUC.GetByID(c.Context(), companyID, id)
	if err != nil {
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "ticket no encontrado"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// UpdateTicket actualiza un ticket.
// @Summary      Actualizar ticket PQR
// @Description  Actualiza los datos o el estado de un ticket PQR existente
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "Ticket ID"
// @Param        body  body      dto.UpdateTicketRequest  true  "Updates"
// @Success      200   {object}  dto.TicketResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /api/crm/tickets/{id} [put]
func (h *CRMHandler) UpdateTicket(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	id := c.Params("id")
	if companyID == "" || id == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.UpdateTicketRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.PQRUC.Update(c.Context(), companyID, id, in)
	if err != nil {
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "ticket no encontrado"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// ListTickets lista tickets de la empresa.
// @Summary      Listar tickets PQR
// @Description  Lista los tickets PQR de la empresa con paginación
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        limit  query     int  false "Limit"
// @Param        offset query     int  false "Offset"
// @Success      200    {object}  dto.TicketResponseList
// @Router       /api/crm/tickets [get]
func (h *CRMHandler) ListTickets(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	out, err := h.PQRUC.ListByCompany(c.Context(), companyID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// GenerateCampaignCopy genera copy de campaña con IA.
// @Summary      Generar copy de campaña con IA
// @Description  Genera textos de campañas de marketing personalizados usando IA
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        body  body      object  true  "{\"prompt\": \"...\"}"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  dto.ErrorResponse
// @Router       /api/crm/ai/campaign-copy [post]
func (h *CRMHandler) GenerateCampaignCopy(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var body struct {
		Prompt string `json:"prompt"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	text, err := h.AICRMUC.GenerateCampaignCopy(c.Context(), body.Prompt)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(fiber.Map{"text": text})
}

// SummarizeTimeline resume el timeline de interacciones de un cliente con IA.
// @Summary      Resumir timeline de interacciones con IA
// @Description  Resume el historial de interacciones de un cliente usando IA
// @Tags         crm
// @Accept       json
// @Produce      json
// @Param        body  body      object  true  "{\"customer_id\": \"...\"}"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  dto.ErrorResponse
// @Router       /api/crm/ai/summarize-timeline [post]
func (h *CRMHandler) SummarizeTimeline(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var body struct {
		CustomerID string `json:"customer_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.CustomerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "customer_id requerido"})
	}
	list, err := h.InteractionRepo.ListByCustomer(body.CustomerID, 100, 0)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	text, err := h.AICRMUC.SummarizeTimeline(c.Context(), list)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(fiber.Map{"summary": text})
}
