package billing

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
)

// CompanyRepository define las operaciones de consulta para Company.
// Inyectado por el caller (ej: main.go)
type CompanyRepository interface {
	GetByID(id string) (*entity.Company, error)
}

// DIANConfigMiddleware inyecta la URL y certificado de DIAN según el environment
// de la empresa obtenida desde el contexto (company_id).
//
// Requiere que:
// - El header Authorization (JWT) haya sido procesado previamente
// - El contexto contenga "company_id" (desde el JWT o middleware de auth)
// - CompanyRepository esté disponible para consultar la empresa
//
// Inyecta en el contexto Fiber:
// - "dian_url": URL del endpoint SOAP de DIAN
// - "dian_cert": Certificado para autenticación
// - "company": La entidad Company completa
func DIANConfigMiddleware(companyRepo CompanyRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyID, ok := c.Locals("company_id").(string)
		if !ok || companyID == "" {
			// Sin company_id, usar defaults
			c.Locals("dian_url", "https://vpfe-hab.dian.gov.co/WcfDianCustomerServices.svc")
			c.Locals("dian_cert", "")
			return c.Next()
		}

		// Obtener la empresa de la base de datos
		company, err := companyRepo.GetByID(companyID)
		if err != nil {
			// Si falla, usar defaults y continuar
			c.Locals("dian_url", "https://vpfe-hab.dian.gov.co/WcfDianCustomerServices.svc")
			c.Locals("dian_cert", "")
			return c.Next()
		}

		// Obtener configuración DIAN de la empresa
		dianURL, dianCert := company.DianConfig()
		c.Locals("dian_url", dianURL)
		c.Locals("dian_cert", dianCert)
		c.Locals("company", company)

		return c.Next()
	}
}

// GetDIANConfigFromContext extrae la configuración inyectada por DIANConfigMiddleware.
// Retorna (url, cert) seguros para pasar a GetAcquirer.
func GetDIANConfigFromContext(c *fiber.Ctx) (url, cert string, err error) {
	url, ok := c.Locals("dian_url").(string)
	if !ok {
		return "", "", fmt.Errorf("dian_url no encontrada en contexto")
	}
	cert, _ = c.Locals("dian_cert").(string)
	return url, cert, nil
}

// GetCompanyFromContext extrae la entidad Company inyectada por DIANConfigMiddleware.
func GetCompanyFromContext(c *fiber.Ctx) (*entity.Company, error) {
	company, ok := c.Locals("company").(*entity.Company)
	if !ok {
		return nil, fmt.Errorf("company no encontrada en contexto")
	}
	return company, nil
}
