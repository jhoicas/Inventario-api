package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
)

type fakeSupplierUseCase struct {
	createFunc     func(companyID string, in dto.CreateSupplierRequest) (*dto.SupplierResponse, error)
	getByIDFunc    func(id string) (*dto.SupplierResponse, error)
	listFunc       func(companyID string, filters dto.SupplierFilters) (*dto.SupplierListResponse, error)
	updateFunc     func(id string, in dto.UpdateSupplierRequest) (*dto.SupplierResponse, error)
	deactivateFunc func(companyID, supplierID string) error
}

func (f *fakeSupplierUseCase) Create(companyID string, in dto.CreateSupplierRequest) (*dto.SupplierResponse, error) {
	if f.createFunc != nil {
		return f.createFunc(companyID, in)
	}
	return nil, errors.New("Create not configured")
}

func (f *fakeSupplierUseCase) GetByID(id string) (*dto.SupplierResponse, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(id)
	}
	return nil, errors.New("GetByID not configured")
}

func (f *fakeSupplierUseCase) List(companyID string, filters dto.SupplierFilters) (*dto.SupplierListResponse, error) {
	if f.listFunc != nil {
		return f.listFunc(companyID, filters)
	}
	return nil, errors.New("List not configured")
}

func (f *fakeSupplierUseCase) Update(id string, in dto.UpdateSupplierRequest) (*dto.SupplierResponse, error) {
	if f.updateFunc != nil {
		return f.updateFunc(id, in)
	}
	return nil, errors.New("Update not configured")
}

func (f *fakeSupplierUseCase) Deactivate(companyID, supplierID string) error {
	if f.deactivateFunc != nil {
		return f.deactivateFunc(companyID, supplierID)
	}
	return errors.New("Deactivate not configured")
}

func TestSupplierHandler_List_TotalInvariantAcrossPagination(t *testing.T) {
	companyID := "comp-sup-1"
	totalBySearch := map[string]int{"acme": 14}

	uc := &fakeSupplierUseCase{
		listFunc: func(inCompanyID string, filters dto.SupplierFilters) (*dto.SupplierListResponse, error) {
			require.Equal(t, companyID, inCompanyID)
			total := totalBySearch[filters.Search]
			items := []dto.SupplierResponse{}
			if filters.Offset < total {
				items = append(items, dto.SupplierResponse{ID: "sup-1", CompanyID: inCompanyID, Name: "ACME"})
			}
			return &dto.SupplierListResponse{Items: items, Total: total, Limit: filters.Limit, Offset: filters.Offset}, nil
		},
	}

	h := NewSupplierHandler(uc)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(LocalCompanyID, companyID)
		return c.Next()
	})
	app.Get("/suppliers", h.List)

	req1 := httptest.NewRequest(http.MethodGet, "/suppliers?search=acme&limit=10&offset=0", nil)
	resp1, err := app.Test(req1, -1)
	require.NoError(t, err)
	defer resp1.Body.Close()
	require.Equal(t, http.StatusOK, resp1.StatusCode)

	var out1 dto.SupplierListResponse
	require.NoError(t, json.NewDecoder(resp1.Body).Decode(&out1))
	assert.Equal(t, 14, out1.Total)
	assert.Equal(t, 10, out1.Limit)
	assert.Equal(t, 0, out1.Offset)

	req2 := httptest.NewRequest(http.MethodGet, "/suppliers?search=acme&limit=3&offset=9", nil)
	resp2, err := app.Test(req2, -1)
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var out2 dto.SupplierListResponse
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&out2))
	assert.Equal(t, 14, out2.Total)
	assert.Equal(t, 3, out2.Limit)
	assert.Equal(t, 9, out2.Offset)

	assert.Equal(t, out1.Total, out2.Total)
}
