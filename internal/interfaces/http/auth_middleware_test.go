package http_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apphttp "github.com/jhoicas/Inventario-api/internal/interfaces/http"
	pkgjwt "github.com/jhoicas/Inventario-api/pkg/jwt"
)

// ──────────────────────────────────────────────────────────────────────────────
// Helpers de test
// ──────────────────────────────────────────────────────────────────────────────

const (
	testJWTSecret = "test-secret-key-for-unit-tests"
	testUserID    = "00000000-0000-0000-0000-000000000001"
	testCompanyID = "00000000-0000-0000-0000-000000000002"
	testIssuer    = "inventory-pro-test"
	testExpMin    = 60
)

// buildTestApp construye una aplicación Fiber mínima con:
//   - AuthMiddleware para parsear el JWT y cargar locals
//   - RequireRole para autorizar el acceso
//   - Un handler dummy que devuelve 200 si pasa los middlewares
func buildTestApp(allowedRoles ...string) *fiber.App {
	app := fiber.New(fiber.Config{
		// Silenciar errores internos en los tests
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})
	// Ruta protegida: JWT + RBAC
	app.Get("/protected",
		apphttp.AuthMiddleware(testJWTSecret),
		apphttp.RequireRole(allowedRoles...),
		func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"ok":   true,
				"role": apphttp.GetRole(c),
			})
		},
	)
	return app
}

// tokenForRole genera un JWT con el rol indicado.
func tokenForRole(t *testing.T, role string) string {
	t.Helper()
	tok, err := pkgjwt.Generate(testJWTSecret, testUserID, testCompanyID, role, testIssuer, testExpMin)
	require.NoError(t, err, "debe generarse un token JWT válido")
	return "Bearer " + tok
}

// doRequest lanza una petición GET /protected y devuelve la respuesta.
func doRequest(t *testing.T, app *fiber.App, authHeader string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	return resp
}

// ──────────────────────────────────────────────────────────────────────────────
// Tests RequireRole
// ──────────────────────────────────────────────────────────────────────────────

// Caso 1: El usuario tiene el rol requerido → debe pasar (HTTP 200).
func TestRequireRole_AdminAccedeRutaAdmin(t *testing.T) {
	app := buildTestApp("admin")
	resp := doRequest(t, app, tokenForRole(t, "admin"))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"admin debe poder acceder a ruta restringida a admin")

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, true, body["ok"], "la respuesta debe incluir ok:true")
	assert.Equal(t, "admin", body["role"], "el role debe ser admin")
}

// Caso 1b: El usuario tiene uno de los roles permitidos (multi-rol) → HTTP 200.
func TestRequireRole_BodegueroAccedeRutaAdminOBodeguero(t *testing.T) {
	app := buildTestApp("admin", "bodeguero")
	resp := doRequest(t, app, tokenForRole(t, "bodeguero"))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"bodeguero debe poder acceder a ruta que permite admin o bodeguero")
}

// Caso 2: El usuario tiene un rol diferente al requerido → HTTP 403 Forbidden.
func TestRequireRole_VendedorBloqueadoEnRutaAdmin(t *testing.T) {
	app := buildTestApp("admin")
	resp := doRequest(t, app, tokenForRole(t, "vendedor"))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode,
		"vendedor no debe poder acceder a ruta restringida a admin")

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "FORBIDDEN",
		"la respuesta de error debe incluir el código FORBIDDEN")
}

// Caso 2b: rol bodeguero bloqueado en ruta solo vendedor → HTTP 403.
func TestRequireRole_BodegueroBloqueadoEnRutaVendedor(t *testing.T) {
	app := buildTestApp("vendedor")
	resp := doRequest(t, app, tokenForRole(t, "bodeguero"))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// Caso 3: Token sin claim de rol (emulado con token vacío) → HTTP 401.
func TestRequireRole_TokenSinRol_Retorna401(t *testing.T) {
	// Generamos un token con rol vacío para simular un token legacy sin el claim.
	app := buildTestApp("admin")
	tok, err := pkgjwt.Generate(testJWTSecret, testUserID, testCompanyID, "", testIssuer, testExpMin)
	require.NoError(t, err)

	resp := doRequest(t, app, "Bearer "+tok)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"token sin rol debe retornar 401")

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "MISSING_ROLE",
		"la respuesta debe indicar el código MISSING_ROLE")
}

// Caso 4: Sin header Authorization → HTTP 401 MISSING_TOKEN.
func TestRequireRole_SinAuthHeader_Retorna401(t *testing.T) {
	app := buildTestApp("admin")
	resp := doRequest(t, app, "") // sin header
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// Caso 5: Token inválido / malformado → HTTP 401 INVALID_TOKEN.
func TestRequireRole_TokenInvalido_Retorna401(t *testing.T) {
	app := buildTestApp("admin")
	resp := doRequest(t, app, "Bearer token.invalido.aqui")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ──────────────────────────────────────────────────────────────────────────────
// Tests AuthMiddleware — extracción de claims del token
// ──────────────────────────────────────────────────────────────────────────────

func TestAuthMiddleware_ExtractaClaims(t *testing.T) {
	app := fiber.New()
	app.Get("/me", apphttp.AuthMiddleware(testJWTSecret), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"user_id":    apphttp.GetUserID(c),
			"company_id": apphttp.GetCompanyID(c),
			"role":       apphttp.GetRole(c),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", tokenForRole(t, "admin"))
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, testUserID, body["user_id"])
	assert.Equal(t, testCompanyID, body["company_id"])
	assert.Equal(t, "admin", body["role"])
}

// ──────────────────────────────────────────────────────────────────────────────
// Tests JWT pkg — integridad del generate/parse con role
// ──────────────────────────────────────────────────────────────────────────────

func TestJWT_GenerateAndParse_ConRole(t *testing.T) {
	tok, err := pkgjwt.Generate(testJWTSecret, testUserID, testCompanyID, "bodeguero", testIssuer, testExpMin)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	userID, companyID, role, err := pkgjwt.Parse(testJWTSecret, tok)
	require.NoError(t, err)

	assert.Equal(t, testUserID, userID)
	assert.Equal(t, testCompanyID, companyID)
	assert.Equal(t, "bodeguero", role)
}

func TestJWT_TokenExpirado_RetornaError(t *testing.T) {
	// Token con expiración -1 minuto (ya expirado)
	tok, err := pkgjwt.Generate(testJWTSecret, testUserID, testCompanyID, "admin", testIssuer, -1)
	require.NoError(t, err)

	_, _, _, err = pkgjwt.Parse(testJWTSecret, tok)
	assert.Error(t, err, "token expirado debe retornar error")
}

func TestJWT_SecretIncorrecto_RetornaError(t *testing.T) {
	tok, err := pkgjwt.Generate(testJWTSecret, testUserID, testCompanyID, "admin", testIssuer, testExpMin)
	require.NoError(t, err)

	_, _, _, err = pkgjwt.Parse("otro-secret-completamente-distinto", tok)
	assert.Error(t, err, "secret incorrecto debe invalidar el token")
}
