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

type fakeAdminUserUseCase struct {
	listByCompanyFunc   func(companyID, search string, limit, offset int) (*dto.UserListResponse, error)
	createForCompanyFun func(companyID string, in dto.AdminCreateUserRequest) (*dto.UserResponse, error)
	updateForCompanyFun func(companyID, userID string, in dto.AdminUpdateUserRequest) (*dto.UserResponse, error)
}

func (f *fakeAdminUserUseCase) ListByCompany(companyID, search string, limit, offset int) (*dto.UserListResponse, error) {
	if f.listByCompanyFunc != nil {
		return f.listByCompanyFunc(companyID, search, limit, offset)
	}
	return nil, errors.New("ListByCompany not configured")
}

func (f *fakeAdminUserUseCase) CreateForCompany(companyID string, in dto.AdminCreateUserRequest) (*dto.UserResponse, error) {
	if f.createForCompanyFun != nil {
		return f.createForCompanyFun(companyID, in)
	}
	return nil, errors.New("CreateForCompany not configured")
}

func (f *fakeAdminUserUseCase) UpdateForCompany(companyID, userID string, in dto.AdminUpdateUserRequest) (*dto.UserResponse, error) {
	if f.updateForCompanyFun != nil {
		return f.updateForCompanyFun(companyID, userID, in)
	}
	return nil, errors.New("UpdateForCompany not configured")
}

func TestAdminUserHandler_ListByCompany_TotalInvariantAcrossPagination(t *testing.T) {
	totalByCompany := map[string]int{"comp-1": 37}

	uc := &fakeAdminUserUseCase{
		listByCompanyFunc: func(companyID, search string, limit, offset int) (*dto.UserListResponse, error) {
			total := totalByCompany[companyID]
			_ = search
			items := []dto.UserResponse{}
			if offset < total {
				items = append(items, dto.UserResponse{ID: "u-1", CompanyID: companyID, Email: "a@acme.com", Name: "A"})
			}
			return &dto.UserListResponse{Items: items, Total: total, Limit: limit, Offset: offset}, nil
		},
	}

	h := NewAdminUserHandler(uc)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/admin/companies/:company_id/users", h.ListByCompany)

	req1 := httptest.NewRequest(http.MethodGet, "/admin/companies/comp-1/users?limit=5&offset=0", nil)
	resp1, err := app.Test(req1, -1)
	require.NoError(t, err)
	defer resp1.Body.Close()
	require.Equal(t, http.StatusOK, resp1.StatusCode)

	var out1 dto.UserListResponse
	require.NoError(t, json.NewDecoder(resp1.Body).Decode(&out1))
	assert.Equal(t, 37, out1.Total)
	assert.Equal(t, 5, out1.Limit)
	assert.Equal(t, 0, out1.Offset)

	req2 := httptest.NewRequest(http.MethodGet, "/admin/companies/comp-1/users?limit=1&offset=20", nil)
	resp2, err := app.Test(req2, -1)
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var out2 dto.UserListResponse
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&out2))
	assert.Equal(t, 37, out2.Total)
	assert.Equal(t, 1, out2.Limit)
	assert.Equal(t, 20, out2.Offset)

	assert.Equal(t, out1.Total, out2.Total)
}
