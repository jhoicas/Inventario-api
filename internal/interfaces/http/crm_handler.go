package http

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/crm"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// invoiceHistoryRepo es la interfaz mínima de InvoiceRepository que necesita CRMHandler.
type invoiceHistoryRepo interface {
	ListByCustomer(customerID string, limit, offset int) ([]*entity.Invoice, int64, error)
	GetCustomerStats(customerID string) (*repository.CustomerPurchaseStats, error)
}

// CRMHandler maneja las peticiones HTTP del módulo CRM (protegido + RequireModule crm).
type CRMHandler struct {
	LoyaltyUC       *crm.LoyaltyUseCase
	TaskUC          *crm.TaskUseCase
	PQRUC           *crm.PQRUseCase
	AICRMUC         *crm.AICRMUseCase
	OpportunityUC   *crm.OpportunityUseCase
	CampaignUC      *crm.CampaignUseCase
	InvoiceHistory  invoiceHistoryRepo
	InteractionRepo interface {
		Create(interaction *entity.CRMInteraction) error
		ListByCustomer(customerID string, limit, offset int) ([]*entity.CRMInteraction, error)
		ListInteractions(customerID string, f repository.InteractionFilters) ([]*entity.CRMInteraction, int64, error)
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
		ListInteractions(customerID string, f repository.InteractionFilters) ([]*entity.CRMInteraction, int64, error)
	},
	opportunityUC *crm.OpportunityUseCase,
	invoiceHistory invoiceHistoryRepo,
	campaignUC *crm.CampaignUseCase,
) *CRMHandler {
	return &CRMHandler{
		LoyaltyUC:       loyaltyUC,
		TaskUC:          taskUC,
		PQRUC:           pqrUC,
		AICRMUC:         aiCRMUC,
		OpportunityUC:   opportunityUC,
		CampaignUC:      campaignUC,
		InvoiceHistory:  invoiceHistory,
		InteractionRepo: interactionRepo,
	}
}

// GetProfile360 obtiene la vista 360 del cliente.
// @Summary      Vista 360 del cliente
// @Description  Obtiene la vista 360 del cliente con datos base, perfil CRM y categoría de fidelización
// @Tags         crm
// @Security     Bearer
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
// @Security     Bearer
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
// @Security     Bearer
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
// @Security     Bearer
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

// CreateBenefit crea un beneficio dentro de una categoría (solo admin).
// @Summary      Crear beneficio
// @Description  Crea un beneficio asociado a una categoría de fidelización (solo admin)
// @Tags         crm
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        categoryId  path      string  true  "Category ID"
// @Param        body        body      dto.CreateBenefitRequest  true  "Benefit"
// @Success      201         {object}  dto.BenefitResponse
// @Failure      400         {object}  dto.ErrorResponse
// @Failure      401         {object}  dto.ErrorResponse
// @Failure      403         {object}  dto.ErrorResponse
// @Failure      404         {object}  dto.ErrorResponse
// @Router       /api/crm/categories/{categoryId}/benefits [post]
func (h *CRMHandler) CreateBenefit(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	categoryID := c.Params("categoryId")
	if companyID == "" || categoryID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.CreateBenefitRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.LoyaltyUC.CreateBenefit(c.Context(), companyID, categoryID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "name requerido"})
		}
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "categoría no encontrada"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// UpdateBenefit actualiza un beneficio (solo admin).
// @Summary      Actualizar beneficio
// @Description  Actualiza un beneficio de fidelización (solo admin)
// @Tags         crm
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        benefitId  path      string  true  "Benefit ID"
// @Param        body       body      dto.UpdateBenefitRequest  true  "Benefit"
// @Success      200        {object}  dto.BenefitResponse
// @Failure      400        {object}  dto.ErrorResponse
// @Failure      401        {object}  dto.ErrorResponse
// @Failure      403        {object}  dto.ErrorResponse
// @Failure      404        {object}  dto.ErrorResponse
// @Router       /api/crm/benefits/{benefitId} [put]
func (h *CRMHandler) UpdateBenefit(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	benefitID := c.Params("benefitId")
	if companyID == "" || benefitID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.UpdateBenefitRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.LoyaltyUC.UpdateBenefit(c.Context(), companyID, benefitID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "name requerido"})
		}
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "beneficio no encontrado"})
		}
		if err == domain.ErrForbidden {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// CreateTask crea una tarea.
// @Summary      Crear tarea CRM
// @Description  Crea una tarea de seguimiento o gestión comercial para un cliente
// @Tags         crm
// @Security     Bearer
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
// @Security     Bearer
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
// @Security     Bearer
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
// @Security     Bearer
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
// @Security     Bearer
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

// ListInteractions lista interacciones de un cliente con filtros opcionales.
// @Summary      Listar interacciones por cliente
// @Description  Lista interacciones CRM de un cliente con filtros por tipo y rango de fecha
// @Tags         crm
// @Security     Bearer
// @Produce      json
// @Param        id          path   string  true   "Customer ID"
// @Param        type        query  string  false  "Tipo: call|email|meeting|other"
// @Param        start_date  query  string  false  "Fecha inicio RFC3339"
// @Param        end_date    query  string  false  "Fecha fin RFC3339"
// @Param        limit       query  int     false  "Límite (máx 100)"
// @Param        offset      query  int     false  "Offset"
// @Success      200  {object}  dto.InteractionListResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Router       /api/crm/customers/{id}/interactions [get]
func (h *CRMHandler) ListInteractions(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	customerID := c.Params("id")
	if companyID == "" || customerID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}

	interactionType := c.Query("type")
	if interactionType != "" {
		typ := entity.InteractionType(interactionType)
		if typ != entity.InteractionTypeCall && typ != entity.InteractionTypeEmail && typ != entity.InteractionTypeMeeting && typ != entity.InteractionTypeOther {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "type debe ser call, email, meeting u other"})
		}
	}

	var startDate time.Time
	if raw := c.Query("start_date"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "start_date debe estar en formato RFC3339"})
		}
		startDate = parsed
	}

	var endDate time.Time
	if raw := c.Query("end_date"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "end_date debe estar en formato RFC3339"})
		}
		endDate = parsed
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	items, total, err := h.InteractionRepo.ListInteractions(customerID, repository.InteractionFilters{
		Type:      interactionType,
		StartDate: startDate,
		EndDate:   endDate,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	out := make([]dto.InteractionResponse, 0, len(items))
	for _, m := range items {
		out = append(out, dto.InteractionResponse{
			ID:         m.ID,
			CompanyID:  m.CompanyID,
			CustomerID: m.CustomerID,
			Type:       string(m.Type),
			Subject:    m.Subject,
			Body:       m.Body,
			CreatedBy:  m.CreatedBy,
			CreatedAt:  m.CreatedAt,
		})
	}

	return c.JSON(dto.InteractionListResponse{
		Items: out,
		Total: total,
	})
}

// CreateTicket radica un ticket PQR.
// @Summary      Radicar ticket PQR
// @Description  Radica un nuevo caso PQR asociado a un cliente y analiza su sentimiento
// @Tags         crm
// @Security     Bearer
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
// @Security     Bearer
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
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "Ticket ID"
// @Param        body  body      dto.UpdateTicketRequest  true  "Updates"
// @Success      200   {object}  dto.TicketResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /api/crm/tickets/{id} [put]
func (h *CRMHandler) UpdateTicket(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	id := c.Params("id")
	if companyID == "" || userID == "" || id == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var in dto.UpdateTicketRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.PQRUC.Update(c.Context(), companyID, userID, id, in)
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
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        search  query     string  false "Buscar por asunto (subject)"
// @Param        limit  query     int  false "Limit"
// @Param        offset query     int  false "Offset"
// @Param        status query     string false "Filtrar por status"
// @Param        sort   query     string false "Orden por created_at: asc|desc"
// @Success      200    {object}  dto.TicketResponseList
// @Router       /api/crm/tickets [get]
func (h *CRMHandler) ListTickets(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	search := c.Query("search")
	status := c.Query("status")
	sort := c.Query("sort")

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	out, err := h.PQRUC.ListByCompany(c.Context(), companyID, search, status, sort, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// CreateOpportunity crea una oportunidad CRM.
// @Summary      Crear oportunidad
// @Description  Crea una oportunidad de negocio en el embudo de ventas
// @Tags         crm
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateOpportunityRequest  true  "Opportunity"
// @Success      201   {object}  dto.OpportunityResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      503   {object}  dto.ErrorResponse
// @Router       /api/crm/opportunities [post]
func (h *CRMHandler) CreateOpportunity(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.OpportunityUC == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "SERVICE_UNAVAILABLE", Message: "opportunity no configurado"})
	}
	var in dto.CreateOpportunityRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.OpportunityUC.Create(c.Context(), companyID, userID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "title requerido"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// UpdateOpportunityStage actualiza la etapa de una oportunidad.
// @Summary      Actualizar etapa de oportunidad
// @Description  Cambia la etapa del embudo de ventas de una oportunidad
// @Tags         crm
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "Opportunity ID"
// @Param        body  body      object  true  "{\"stage\": \"prospecto|calificado|propuesta|negociacion|ganado|perdido\"}"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Router       /api/crm/opportunities/{id}/stage [put]
func (h *CRMHandler) UpdateOpportunityStage(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	id := c.Params("id")
	if companyID == "" || id == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.OpportunityUC == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "SERVICE_UNAVAILABLE", Message: "opportunity no configurado"})
	}
	var body struct {
		Stage string `json:"stage"`
	}
	if err := c.BodyParser(&body); err != nil || body.Stage == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "stage requerido"})
	}
	err := h.OpportunityUC.UpdateStage(c.Context(), companyID, id, body.Stage)
	if err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "stage inválido"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "oportunidad no encontrada"})
		case domain.ErrForbidden:
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// GetOpportunityFunnel retorna el embudo de ventas por etapa.
// @Summary      Embudo de ventas
// @Description  Retorna el conteo y monto total de oportunidades agrupadas por etapa
// @Tags         crm
// @Security     Bearer
// @Produce      json
// @Success      200   {array}   dto.FunnelStageDTO
// @Failure      503   {object}  dto.ErrorResponse
// @Router       /api/crm/opportunities/funnel [get]
func (h *CRMHandler) GetOpportunityFunnel(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.OpportunityUC == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "SERVICE_UNAVAILABLE", Message: "opportunity no configurado"})
	}
	out, err := h.OpportunityUC.GetFunnel(c.Context(), companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// GetPurchaseHistory retorna el historial de compras y estadísticas de un cliente.
// Solo disponible cuando el módulo billing está activo (garantizado por el router).
// @Summary      Historial de compras del cliente
// @Description  Devuelve estadísticas agregadas y lista de facturas paginadas del cliente
// @Tags         crm
// @Security     Bearer
// @Produce      json
// @Param        id      path   string  true  "Customer ID"
// @Param        limit   query  int     false "Límite de facturas a retornar (máx 100)"
// @Param        offset  query  int     false "Offset para paginación"
// @Success      200  {object}  dto.PurchaseHistoryResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      503  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/crm/customers/{id}/purchase-history [get]
func (h *CRMHandler) GetPurchaseHistory(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	customerID := c.Params("id")
	if companyID == "" || customerID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.InvoiceHistory == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "SERVICE_UNAVAILABLE", Message: "módulo billing no disponible"})
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

	stats, err := h.InvoiceHistory.GetCustomerStats(customerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	invoices, total, err := h.InvoiceHistory.ListByCustomer(customerID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}

	summaries := make([]dto.InvoiceSummaryDTO, 0, len(invoices))
	for _, inv := range invoices {
		summaries = append(summaries, dto.InvoiceSummaryDTO{
			ID:           inv.ID,
			Prefix:       inv.Prefix,
			Number:       inv.Number,
			Date:         inv.Date.Format(time.RFC3339),
			GrandTotal:   inv.GrandTotal,
			DocumentType: inv.DocumentType,
			DIANStatus:   inv.DIAN_Status,
		})
	}

	var lastPurchaseStr string
	if !stats.LastPurchaseDate.IsZero() {
		lastPurchaseStr = stats.LastPurchaseDate.Format(time.RFC3339)
	}

	return c.JSON(dto.PurchaseHistoryResponse{
		Stats: dto.CustomerPurchaseStatsDTO{
			TotalPurchases:   stats.TotalPurchases,
			AvgTicket:        stats.AvgTicket,
			LastPurchaseDate: lastPurchaseStr,
			InvoiceCount:     stats.InvoiceCount,
		},
		Invoices: summaries,
		Total:    total,
	})
}

// GenerateCampaignCopy genera copy de campaña con IA.
// @Summary      Generar copy de campaña con IA
// @Description  Genera textos de campañas de marketing personalizados usando IA
// @Tags         crm
// @Security     Bearer
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

// CreateCampaign crea una campaña de marketing CRM.
// @Summary      Crear campaña
// @Description  Crea una campaña de marketing en estado BORRADOR
// @Tags         crm
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateCampaignRequest  true  "Campaign"
// @Success      201   {object}  dto.CampaignResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      503   {object}  dto.ErrorResponse
// @Router       /api/crm/campaigns [post]
func (h *CRMHandler) CreateCampaign(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	userID := GetUserID(c)
	if companyID == "" || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.CampaignUC == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "SERVICE_UNAVAILABLE", Message: "campaigns no configurado"})
	}
	var in dto.CreateCampaignRequest
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "INVALID_BODY", Message: "cuerpo inválido"})
	}
	out, err := h.CampaignUC.Create(c.Context(), companyID, userID, in)
	if err != nil {
		if err == domain.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "name requerido"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// GetCampaignMetrics retorna las métricas de una campaña.
// @Summary      Métricas de campaña
// @Description  Devuelve contadores de envío, apertura, clics, conversión e ingresos de una campaña
// @Tags         crm
// @Security     Bearer
// @Produce      json
// @Param        id  path  string  true  "Campaign ID"
// @Success      200  {object}  dto.CampaignMetricsResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      503  {object}  dto.ErrorResponse
// @Router       /api/crm/campaigns/{id}/metrics [get]
func (h *CRMHandler) GetCampaignMetrics(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	id := c.Params("id")
	if companyID == "" || id == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	if h.CampaignUC == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(dto.ErrorResponse{Code: "SERVICE_UNAVAILABLE", Message: "campaigns no configurado"})
	}
	out, err := h.CampaignUC.GetMetrics(c.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "campaña no encontrada"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(out)
}

// EscalateTicket escala un ticket PQR y registra la razón.
// @Summary      Escalar ticket PQR
// @Description  Marca el ticket como ESCALATED, persiste la razón y genera una entrada de auditoría
// @Tags         crm
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "Ticket ID"
// @Param        body  body      object  true  "{\"reason\": \"...\"}"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      403   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Router       /api/crm/tickets/{id}/escalate [put]
func (h *CRMHandler) EscalateTicket(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	ticketID := c.Params("id")
	if companyID == "" || ticketID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil || body.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "reason requerido"})
	}
	if err := h.PQRUC.EscalateTicket(c.Context(), companyID, ticketID, body.Reason); err != nil {
		switch err {
		case domain.ErrInvalidInput:
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Code: "VALIDATION", Message: "parámetros inválidos"})
		case domain.ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{Code: "NOT_FOUND", Message: "ticket no encontrado"})
		case domain.ErrForbidden:
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse{Code: "FORBIDDEN", Message: "acceso denegado"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	return c.JSON(fiber.Map{"status": "escalated"})
}

// ListOverdueTickets lista los tickets en estado OVERDUE de la empresa.
// @Summary      Tickets vencidos (OVERDUE)
// @Description  Devuelve los tickets cuyo SLA ha expirado y fueron marcados como OVERDUE
// @Tags         crm
// @Security     Bearer
// @Produce      json
// @Success      200  {array}   dto.TicketResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Router       /api/crm/tickets/overdue [get]
func (h *CRMHandler) ListOverdueTickets(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse{Code: "UNAUTHORIZED", Message: "token inválido"})
	}
	list, err := h.PQRUC.ListOverdue(c.Context(), companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{Code: "INTERNAL", Message: err.Error()})
	}
	items := make([]dto.TicketResponse, 0, len(list))
	for _, t := range list {
		items = append(items, dto.TicketResponse{
			ID:               t.ID,
			CompanyID:        t.CompanyID,
			CustomerID:       t.CustomerID,
			Subject:          t.Subject,
			Description:      t.Description,
			Status:           t.Status,
			Sentiment:        t.Sentiment,
			EscalationReason: t.EscalationReason,
			CreatedBy:        t.CreatedBy,
			CreatedAt:        t.CreatedAt,
			UpdatedAt:        t.UpdatedAt,
		})
	}
	return c.JSON(items)
}

// SummarizeTimeline resume el timeline de interacciones de un cliente con IA.
// @Summary      Resumir timeline de interacciones con IA
// @Description  Resume el historial de interacciones de un cliente usando IA
// @Tags         crm
// @Security     Bearer
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
