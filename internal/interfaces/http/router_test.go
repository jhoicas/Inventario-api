package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jhoicas/Inventario-api/internal/application/usecase"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgjwt "github.com/jhoicas/Inventario-api/pkg/jwt"
)

const (
	routerTestJWTSecret = "router-test-secret"
	routerTestUserID    = "00000000-0000-0000-0000-000000000011"
	routerTestCompanyID = "00000000-0000-0000-0000-000000000022"
	routerTestIssuer    = "router-test"
)

type fakeCompanyRepoForRouter struct {
	hasActiveModuleFunc func(ctx context.Context, companyID, moduleName string) (bool, error)
}

func (f *fakeCompanyRepoForRouter) Create(company *entity.Company) error         { return nil }
func (f *fakeCompanyRepoForRouter) GetByID(id string) (*entity.Company, error)   { return nil, nil }
func (f *fakeCompanyRepoForRouter) GetByNIT(nit string) (*entity.Company, error) { return nil, nil }
func (f *fakeCompanyRepoForRouter) Update(company *entity.Company) error         { return nil }
func (f *fakeCompanyRepoForRouter) List(limit, offset int) ([]*entity.Company, error) {
	return nil, nil
}
func (f *fakeCompanyRepoForRouter) Delete(id string) error { return nil }
func (f *fakeCompanyRepoForRouter) ListModules(ctx context.Context, companyID string) ([]*entity.CompanyModule, error) {
	return []*entity.CompanyModule{}, nil
}
func (f *fakeCompanyRepoForRouter) GetModule(ctx context.Context, companyID, moduleName string) (*entity.CompanyModule, error) {
	return nil, nil
}
func (f *fakeCompanyRepoForRouter) UpsertModule(ctx context.Context, module *entity.CompanyModule) error {
	return nil
}
func (f *fakeCompanyRepoForRouter) DeleteModule(ctx context.Context, companyID, moduleName string) error {
	return nil
}
func (f *fakeCompanyRepoForRouter) HasActiveModule(ctx context.Context, companyID, moduleName string) (bool, error) {
	if f.hasActiveModuleFunc != nil {
		return f.hasActiveModuleFunc(ctx, companyID, moduleName)
	}
	return false, nil
}

func tokenForRouterRole(t *testing.T, role string) string {
	t.Helper()
	tok, err := pkgjwt.Generate(routerTestJWTSecret, routerTestUserID, routerTestCompanyID, []string{role}, routerTestIssuer, 60)
	require.NoError(t, err)
	return "Bearer " + tok
}

func TestRouter_ReorderConfigRouteProtections(t *testing.T) {
	body := `{"warehouse_id":"wh-1","reorder_point":"10","min_stock":"2","max_stock":"30","lead_time_days":5}`

	tests := []struct {
		name           string
		authHeader     string
		moduleActive   bool
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Unauthorized_NoToken",
			authHeader:     "",
			moduleActive:   true,
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "MISSING_TOKEN",
		},
		{
			name:           "Forbidden_ModuleDisabled",
			authHeader:     tokenForRouterRole(t, entity.RoleAdmin),
			moduleActive:   false,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "MODULE_DISABLED",
		},
		{
			name:           "Forbidden_RoleNotAllowed",
			authHeader:     tokenForRouterRole(t, entity.RoleVendedor),
			moduleActive:   true,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name:           "PassesMiddlewares_ReachesHandler",
			authHeader:     tokenForRouterRole(t, entity.RoleAdmin),
			moduleActive:   true,
			expectedStatus: http.StatusServiceUnavailable,
			expectedCode:   "SERVICE_UNAVAILABLE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			companyRepo := &fakeCompanyRepoForRouter{hasActiveModuleFunc: func(ctx context.Context, companyID, moduleName string) (bool, error) {
				if companyID != routerTestCompanyID {
					return false, nil
				}
				if moduleName != entity.ModuleInventory {
					return false, nil
				}
				return tt.moduleActive, nil
			}}

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			Router(app, RouterDeps{
				ModuleService: usecase.NewModuleService(companyRepo),
				JWTSecret:     routerTestJWTSecret,
			})

			req := httptest.NewRequest(http.MethodPut, "/api/products/prod-1/reorder-config", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var out map[string]any
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
			assert.Equal(t, tt.expectedCode, out["code"])
		})
	}
}

func TestRouter_ReorderConfigRoutePathExists(t *testing.T) {
	companyRepo := &fakeCompanyRepoForRouter{hasActiveModuleFunc: func(ctx context.Context, companyID, moduleName string) (bool, error) {
		return true, nil
	}}

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	Router(app, RouterDeps{
		ModuleService: usecase.NewModuleService(companyRepo),
		JWTSecret:     routerTestJWTSecret,
	})

	tok := tokenForRouterRole(t, entity.RoleAdmin)
	req := httptest.NewRequest(http.MethodPut, "/api/products/prod-xyz/reorder-config", strings.NewReader(`{"warehouse_id":"wh-1","reorder_point":"10","min_stock":"1","max_stock":"20","lead_time_days":3}`))
	req.Header.Set("Authorization", tok)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Si la ruta no existiera devolvería 404; aquí esperamos llegar al handler y obtener 503 por UC no configurado.
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestRouter_ReorderConfigRequestBodyDecimals_Parse(t *testing.T) {
	companyRepo := &fakeCompanyRepoForRouter{hasActiveModuleFunc: func(ctx context.Context, companyID, moduleName string) (bool, error) {
		return true, nil
	}}

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	Router(app, RouterDeps{
		ModuleService: usecase.NewModuleService(companyRepo),
		JWTSecret:     routerTestJWTSecret,
	})

	tok := tokenForRouterRole(t, entity.RoleAdmin)
	payload := struct {
		WarehouseID  string          `json:"warehouse_id"`
		ReorderPoint decimal.Decimal `json:"reorder_point"`
		MinStock     decimal.Decimal `json:"min_stock"`
		MaxStock     decimal.Decimal `json:"max_stock"`
		LeadTimeDays int             `json:"lead_time_days"`
	}{
		WarehouseID:  "wh-1",
		ReorderPoint: decimal.NewFromInt(10),
		MinStock:     decimal.NewFromInt(2),
		MaxStock:     decimal.NewFromInt(30),
		LeadTimeDays: 5,
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/products/prod-1/reorder-config", strings.NewReader(string(b)))
	req.Header.Set("Authorization", tok)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// La petición pasa parseo y middlewares; falla recién en handler por UC no configurado.
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestRouter_StocktakeRoutesProtections(t *testing.T) {
	type endpointCase struct {
		name   string
		method string
		path   string
		body   string
	}

	endpoints := []endpointCase{
		{name: "CreateSnapshot", method: http.MethodPost, path: "/api/inventory/stocktake", body: `{"warehouse_id":"wh-1"}`},
		{name: "UpdateCounts", method: http.MethodPut, path: "/api/inventory/stocktake/st-1", body: `{"items":[{"product_id":"p-1","counted_qty":"10"}]}`},
		{name: "Close", method: http.MethodPost, path: "/api/inventory/stocktake/st-1/close", body: ``},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			t.Run("Unauthorized_NoToken", func(t *testing.T) {
				companyRepo := &fakeCompanyRepoForRouter{hasActiveModuleFunc: func(ctx context.Context, companyID, moduleName string) (bool, error) {
					return true, nil
				}}
				app := fiber.New(fiber.Config{DisableStartupMessage: true})
				Router(app, RouterDeps{ModuleService: usecase.NewModuleService(companyRepo), JWTSecret: routerTestJWTSecret})

				req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req, -1)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
				var out map[string]any
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "MISSING_TOKEN", out["code"])
			})

			t.Run("Forbidden_ModuleDisabled", func(t *testing.T) {
				companyRepo := &fakeCompanyRepoForRouter{hasActiveModuleFunc: func(ctx context.Context, companyID, moduleName string) (bool, error) {
					if moduleName == entity.ModuleInventory {
						return false, nil
					}
					return true, nil
				}}
				app := fiber.New(fiber.Config{DisableStartupMessage: true})
				Router(app, RouterDeps{ModuleService: usecase.NewModuleService(companyRepo), JWTSecret: routerTestJWTSecret})

				req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", tokenForRouterRole(t, entity.RoleAdmin))
				resp, err := app.Test(req, -1)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusForbidden, resp.StatusCode)
				var out map[string]any
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "MODULE_DISABLED", out["code"])
			})

			t.Run("Forbidden_RoleNotAllowed", func(t *testing.T) {
				companyRepo := &fakeCompanyRepoForRouter{hasActiveModuleFunc: func(ctx context.Context, companyID, moduleName string) (bool, error) {
					return true, nil
				}}
				app := fiber.New(fiber.Config{DisableStartupMessage: true})
				Router(app, RouterDeps{ModuleService: usecase.NewModuleService(companyRepo), JWTSecret: routerTestJWTSecret})

				req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", tokenForRouterRole(t, entity.RoleVendedor))
				resp, err := app.Test(req, -1)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusForbidden, resp.StatusCode)
				var out map[string]any
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "FORBIDDEN", out["code"])
			})

			t.Run("PassesMiddlewares_ReachesHandler", func(t *testing.T) {
				companyRepo := &fakeCompanyRepoForRouter{hasActiveModuleFunc: func(ctx context.Context, companyID, moduleName string) (bool, error) {
					return true, nil
				}}
				app := fiber.New(fiber.Config{DisableStartupMessage: true})
				Router(app, RouterDeps{ModuleService: usecase.NewModuleService(companyRepo), JWTSecret: routerTestJWTSecret})

				req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", tokenForRouterRole(t, entity.RoleAdmin))
				resp, err := app.Test(req, -1)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
				var out map[string]any
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "SERVICE_UNAVAILABLE", out["code"])
			})
		})
	}
}

type fakeDIANSettingsRepoForRouter struct{}

func (f *fakeDIANSettingsRepoForRouter) Upsert(ctx context.Context, settings *entity.DIANSettings) error {
	return nil
}

func (f *fakeDIANSettingsRepoForRouter) GetByCompanyID(ctx context.Context, companyID string) (*entity.DIANSettings, error) {
	return nil, nil
}

func (f *fakeDIANSettingsRepoForRouter) GetByCompanyIDAndEnvironment(ctx context.Context, companyID, environment string) (*entity.DIANSettings, error) {
	return nil, nil
}

func TestRouter_DIANSettingsGetRouteProtections(t *testing.T) {
	dianSettingsUC := usecase.NewDIANSettingsUseCase(
		&fakeCompanyRepoForRouter{},
		&fakeDIANSettingsRepoForRouter{},
		nil,
		nil,
	)

	routes := []string{
		"/api/settings/dian",
		"/api/dian/settings",
		"/api/dian/configuration",
	}

	tests := []struct {
		name           string
		authHeader     string
		moduleActive   bool
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Unauthorized_NoToken",
			authHeader:     "",
			moduleActive:   true,
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "MISSING_TOKEN",
		},
		{
			name:           "Forbidden_ModuleDisabled",
			authHeader:     tokenForRouterRole(t, entity.RoleAdmin),
			moduleActive:   false,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "MODULE_DISABLED",
		},
		{
			name:           "Forbidden_RoleNotAllowed",
			authHeader:     tokenForRouterRole(t, entity.RoleVendedor),
			moduleActive:   true,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name:           "PassesMiddlewares_ReachesHandler",
			authHeader:     tokenForRouterRole(t, entity.RoleAdmin),
			moduleActive:   true,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
	}

	for _, route := range routes {
		for _, tt := range tests {
			t.Run(route+"/"+tt.name, func(t *testing.T) {
				companyRepo := &fakeCompanyRepoForRouter{hasActiveModuleFunc: func(ctx context.Context, companyID, moduleName string) (bool, error) {
					if companyID != routerTestCompanyID {
						return false, nil
					}
					if moduleName != entity.ModuleBilling {
						return false, nil
					}
					return tt.moduleActive, nil
				}}

				app := fiber.New(fiber.Config{DisableStartupMessage: true})
				Router(app, RouterDeps{
					ModuleService:  usecase.NewModuleService(companyRepo),
					DIANSettingsUC: dianSettingsUC,
					JWTSecret:      routerTestJWTSecret,
				})

				req := httptest.NewRequest(http.MethodGet, route, nil)
				if tt.authHeader != "" {
					req.Header.Set("Authorization", tt.authHeader)
				}

				resp, err := app.Test(req, -1)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, tt.expectedStatus, resp.StatusCode)
				var out map[string]any
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, tt.expectedCode, out["code"])
			})
		}
	}

	for _, route := range routes {
		t.Run(route+"/RoutePathExists", func(t *testing.T) {
			companyRepo := &fakeCompanyRepoForRouter{hasActiveModuleFunc: func(ctx context.Context, companyID, moduleName string) (bool, error) {
				if moduleName == entity.ModuleBilling {
					return true, nil
				}
				return false, nil
			}}

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			Router(app, RouterDeps{
				ModuleService:  usecase.NewModuleService(companyRepo),
				DIANSettingsUC: dianSettingsUC,
				JWTSecret:      routerTestJWTSecret,
			})

			req := httptest.NewRequest(http.MethodGet, route, nil)
			req.Header.Set("Authorization", tokenForRouterRole(t, entity.RoleAdmin))

			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			var out map[string]any
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
			assert.Equal(t, "NOT_FOUND", out["code"])
		})
	}
}
