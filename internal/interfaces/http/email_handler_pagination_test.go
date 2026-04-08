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

type fakeEmailUseCase struct {
	listAccountsFunc func(companyID string, limit, offset int) (*dto.EmailAccountListResponse, error)
	listEmailsFunc   func(companyID, customerID string, isRead *bool, limit, offset int) (*dto.EmailListResponse, error)
}

func (f *fakeEmailUseCase) CreateAccount(companyID string, in dto.CreateEmailAccountRequest) (*dto.EmailAccountResponse, error) {
	return nil, errors.New("CreateAccount not configured")
}
func (f *fakeEmailUseCase) ProcessOAuthAccount(companyID, userID string, in dto.OAuthEmailAccountRequest) (*dto.EmailAccountConfigResponse, error) {
	return nil, errors.New("ProcessOAuthAccount not configured")
}
func (f *fakeEmailUseCase) SaveGoogleOAuthCredential(companyID, userID string, in dto.GoogleOAuthCredentialRequest) error {
	return errors.New("SaveGoogleOAuthCredential not configured")
}
func (f *fakeEmailUseCase) SaveMicrosoftOAuthCredential(companyID, userID string, in dto.GoogleOAuthCredentialRequest) error {
	return errors.New("SaveMicrosoftOAuthCredential not configured")
}
func (f *fakeEmailUseCase) GetGoogleOAuthCredentialStatus(companyID string) (*dto.GoogleOAuthCredentialStatusResponse, error) {
	return nil, errors.New("GetGoogleOAuthCredentialStatus not configured")
}
func (f *fakeEmailUseCase) SaveCustomAccount(companyID, userID string, in dto.CustomEmailAccountRequest) (*dto.EmailAccountConfigResponse, error) {
	return nil, errors.New("SaveCustomAccount not configured")
}
func (f *fakeEmailUseCase) UpdateAccount(companyID, id string, in dto.UpdateEmailAccountRequest) (*dto.EmailAccountResponse, error) {
	return nil, errors.New("UpdateAccount not configured")
}
func (f *fakeEmailUseCase) DeleteAccount(companyID, id string) error {
	return errors.New("DeleteAccount not configured")
}
func (f *fakeEmailUseCase) GetAccount(companyID, id string) (*dto.EmailAccountResponse, error) {
	return nil, errors.New("GetAccount not configured")
}
func (f *fakeEmailUseCase) ListAccounts(companyID string, limit, offset int) (*dto.EmailAccountListResponse, error) {
	if f.listAccountsFunc != nil {
		return f.listAccountsFunc(companyID, limit, offset)
	}
	return nil, errors.New("ListAccounts not configured")
}
func (f *fakeEmailUseCase) TestConnectionBeforeSave(companyID string, in dto.CreateEmailAccountRequest) (*dto.TestIMAPConnectionResponse, error) {
	return nil, errors.New("TestConnectionBeforeSave not configured")
}
func (f *fakeEmailUseCase) TestConnection(companyID, id string) (*dto.TestIMAPConnectionResponse, error) {
	return nil, errors.New("TestConnection not configured")
}
func (f *fakeEmailUseCase) GetAccountEmails(companyID, accountID string) (*dto.AccountEmailListResponse, error) {
	return nil, errors.New("GetAccountEmails not configured")
}
func (f *fakeEmailUseCase) ListEmails(companyID, customerID string, isRead *bool, limit, offset int) (*dto.EmailListResponse, error) {
	if f.listEmailsFunc != nil {
		return f.listEmailsFunc(companyID, customerID, isRead, limit, offset)
	}
	return nil, errors.New("ListEmails not configured")
}
func (f *fakeEmailUseCase) GetEmailAndMarkAsRead(companyID, id string) (*dto.EmailResponse, error) {
	return nil, errors.New("GetEmailAndMarkAsRead not configured")
}
func (f *fakeEmailUseCase) CreateTicketFromEmail(companyID, userID, emailID string) (*dto.CreateTicketFromEmailResponse, error) {
	return nil, errors.New("CreateTicketFromEmail not configured")
}

func TestEmailHandler_ListEmailAccounts_TotalInvariantAcrossPagination(t *testing.T) {
	companyID := "comp-mail-1"
	uc := &fakeEmailUseCase{
		listAccountsFunc: func(inCompanyID string, limit, offset int) (*dto.EmailAccountListResponse, error) {
			require.Equal(t, companyID, inCompanyID)
			items := []dto.EmailAccountResponse{}
			if offset < 8 {
				items = append(items, dto.EmailAccountResponse{ID: "acc-1", CompanyID: inCompanyID, EmailAddress: "ventas@acme.com"})
			}
			return &dto.EmailAccountListResponse{Items: items, Total: 8, Limit: limit, Offset: offset}, nil
		},
	}

	h := NewEmailHandler(uc)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(LocalCompanyID, companyID)
		return c.Next()
	})
	app.Get("/settings/email-accounts", h.ListEmailAccounts)

	req1 := httptest.NewRequest(http.MethodGet, "/settings/email-accounts?limit=5&offset=0", nil)
	resp1, err := app.Test(req1, -1)
	require.NoError(t, err)
	defer resp1.Body.Close()
	var out1 dto.EmailAccountListResponse
	require.NoError(t, json.NewDecoder(resp1.Body).Decode(&out1))

	req2 := httptest.NewRequest(http.MethodGet, "/settings/email-accounts?limit=2&offset=6", nil)
	resp2, err := app.Test(req2, -1)
	require.NoError(t, err)
	defer resp2.Body.Close()
	var out2 dto.EmailAccountListResponse
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&out2))

	assert.Equal(t, 8, out1.Total)
	assert.Equal(t, 8, out2.Total)
	assert.Equal(t, out1.Total, out2.Total)
}

func TestEmailHandler_ListEmails_TotalInvariantWithFilters(t *testing.T) {
	companyID := "comp-mail-1"
	customerID := "cust-99"

	uc := &fakeEmailUseCase{
		listEmailsFunc: func(inCompanyID, inCustomerID string, isRead *bool, limit, offset int) (*dto.EmailListResponse, error) {
			require.Equal(t, companyID, inCompanyID)
			require.Equal(t, customerID, inCustomerID)
			require.NotNil(t, isRead)
			require.True(t, *isRead)
			items := []dto.EmailResponse{}
			if offset < 21 {
				items = append(items, dto.EmailResponse{ID: "email-1", CustomerID: inCustomerID, Subject: "Hola"})
			}
			return &dto.EmailListResponse{Items: items, Total: 21, Limit: limit, Offset: offset}, nil
		},
	}

	h := NewEmailHandler(uc)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(LocalCompanyID, companyID)
		return c.Next()
	})
	app.Get("/emails", h.ListEmails)

	req1 := httptest.NewRequest(http.MethodGet, "/emails?customer_id=cust-99&is_read=true&limit=10&offset=0", nil)
	resp1, err := app.Test(req1, -1)
	require.NoError(t, err)
	defer resp1.Body.Close()
	var out1 dto.EmailListResponse
	require.NoError(t, json.NewDecoder(resp1.Body).Decode(&out1))

	req2 := httptest.NewRequest(http.MethodGet, "/emails?customer_id=cust-99&is_read=true&limit=3&offset=12", nil)
	resp2, err := app.Test(req2, -1)
	require.NoError(t, err)
	defer resp2.Body.Close()
	var out2 dto.EmailListResponse
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&out2))

	assert.Equal(t, int64(21), out1.Total)
	assert.Equal(t, int64(21), out2.Total)
	assert.Equal(t, out1.Total, out2.Total)
}

func TestEmailHandler_ListEmails_LegacyCompatibilityMode(t *testing.T) {
	t.Setenv("PAGINATION_LEGACY_COMPAT_ENABLED", "true")

	companyID := "comp-mail-1"
	uc := &fakeEmailUseCase{
		listEmailsFunc: func(inCompanyID, inCustomerID string, isRead *bool, limit, offset int) (*dto.EmailListResponse, error) {
			require.Equal(t, companyID, inCompanyID)
			return &dto.EmailListResponse{Items: []dto.EmailResponse{}, Total: 3, Limit: limit, Offset: offset}, nil
		},
	}

	h := NewEmailHandler(uc)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(LocalCompanyID, companyID)
		return c.Next()
	})
	app.Get("/emails", h.ListEmails)

	req := httptest.NewRequest(http.MethodGet, "/emails?legacy=true&limit=2&offset=1", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "legacy", resp.Header.Get("X-Pagination-Format"))

	var out map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	page, ok := out["page"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(3), page["total"])
	assert.Equal(t, float64(2), page["limit"])
	assert.Equal(t, float64(1), page["offset"])
	_, hasTotal := out["total"]
	assert.False(t, hasTotal)
}
