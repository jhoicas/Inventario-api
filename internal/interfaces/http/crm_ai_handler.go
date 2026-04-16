package http

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/ports"
	"github.com/jhoicas/Inventario-api/pkg/logger"
)

type askAIRequest struct {
	Question string `json:"question"`
}

// CRMAIHandler expone endpoints HTTP para analitica IA y carga masiva de ventas.
type CRMAIHandler struct {
	aiAnalyst    ports.AIAnalystService
	bulkImporter ports.BulkImporterService
	log          *logger.Logger
}

// NewCRMAIHandler construye el handler de IA para CRM.
func NewCRMAIHandler(aiAnalyst ports.AIAnalystService, bulkImporter ports.BulkImporterService, log *logger.Logger) *CRMAIHandler {
	return &CRMAIHandler{aiAnalyst: aiAnalyst, bulkImporter: bulkImporter, log: log}
}

// AskAI recibe una pregunta de lenguaje natural y retorna un resumen y datos tabulares.
func (h *CRMAIHandler) AskAI(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "UNAUTHORIZED", "message": "company_id no encontrado en el token"})
	}

	var req askAIRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "INVALID_BODY", "message": "cuerpo inválido"})
	}
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "VALIDATION", "message": "question es requerido"})
	}

	rows, err := h.aiAnalyst.Ask(c.Context(), companyID, req.Question)
	if err != nil {
		if h.log != nil {
			h.log.Error().Err(err).Str("company_id", companyID).Msg("crm_ai.ask failed")
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "INTERNAL", "message": err.Error()})
	}

	summary := fmt.Sprintf("Se encontraron %d registros para la consulta.", len(rows))
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"summary": summary,
		"data":    rows,
	})
}

// ImportSalesFile procesa multipart/form-data con archivo y mappings JSON.
func (h *CRMAIHandler) ImportSalesFile(c *fiber.Ctx) error {
	companyID := GetCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "UNAUTHORIZED", "message": "company_id no encontrado en el token"})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "INVALID_REQUEST", "message": "file es requerido"})
	}

	mappingsRaw := strings.TrimSpace(c.FormValue("mappings"))
	if mappingsRaw == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "VALIDATION", "message": "mappings es requerido"})
	}

	var mappings map[string]interface{}
	if err := json.Unmarshal([]byte(mappingsRaw), &mappings); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "INVALID_MAPPINGS", "message": "mappings debe ser un JSON válido"})
	}

	tableName := mapString(mappings, "table_name", "tableName", "target_table", "targetTable")
	if tableName == "" {
		tableName = "sales"
	}

	sheetName := mapString(mappings, "sheet_name", "sheetName")
	if sheetName == "" {
		sheetName = "Sheet1"
	}

	format := strings.ToLower(mapString(mappings, "format", "file_type", "fileType"))
	if format == "" {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		switch ext {
		case ".csv":
			format = "csv"
		case ".xlsx", ".xls":
			format = "excel"
		}
	}

	tmp, err := os.CreateTemp("", "crm-sales-import-*")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "INTERNAL", "message": "no se pudo preparar archivo temporal"})
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	if err := c.SaveFile(file, tmpPath); err != nil {
		if h.log != nil {
			h.log.Error().Err(err).Str("filename", file.Filename).Msg("crm_ai.import save file failed")
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "INTERNAL", "message": "no se pudo guardar el archivo"})
	}

	var processed int
	switch format {
	case "csv":
		processed, err = h.bulkImporter.ImportFromCSV(c.Context(), companyID, tmpPath, tableName)
	case "excel", "xlsx", "xls":
		processed, err = h.bulkImporter.ImportFromExcel(c.Context(), companyID, tmpPath, sheetName, tableName)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"code": "VALIDATION", "message": "formato no soportado; use csv o excel"})
	}
	if err != nil {
		if h.log != nil {
			h.log.Error().Err(err).Str("company_id", companyID).Str("table", tableName).Msg("crm_ai.import failed")
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "INTERNAL", "message": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":            "ok",
		"processed_records": processed,
	})
}

func mapString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s != "" {
			return s
		}
	}
	return ""
}
