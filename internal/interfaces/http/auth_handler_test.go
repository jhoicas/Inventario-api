package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
)

// ── Fake AuthUseCase (mock manual para tests) ───────────────────────────────────

type fakeAuthUseCase struct {
	registerFunc func(in dto.RegisterRequest) (*dto.UserResponse, error)
	loginFunc    func(in dto.LoginRequest) (*dto.LoginResponse, error)
}

func (f *fakeAuthUseCase) RegisterUser(in dto.RegisterRequest) (*dto.UserResponse, error) {
	if f.registerFunc != nil {
		return f.registerFunc(in)
	}
	return nil, errors.New("registerUser not configured")
}

func (f *fakeAuthUseCase) Login(in dto.LoginRequest) (*dto.LoginResponse, error) {
	if f.loginFunc != nil {
		return f.loginFunc(in)
	}
	return nil, errors.New("login not configured")
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func validLoginRequest() dto.LoginRequest {
	return dto.LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	}
}

func validRegisterRequest() dto.RegisterRequest {
	return dto.RegisterRequest{
		Email:     "user@example.com",
		Password:  "password123",
		CompanyID: "company-123",
		Name:      "Usuario Test",
		Role:      "admin",
	}
}

func validUserResponse() *dto.UserResponse {
	now := time.Now()
	return &dto.UserResponse{
		ID:        "user-123",
		CompanyID: "company-123",
		Email:     "user@example.com",
		Name:      "Usuario Test",
		Role:      "admin",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func validLoginResponse() *dto.LoginResponse {
	return &dto.LoginResponse{
		Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
		User:  *validUserResponse(),
	}
}

// ── Tests Register ──────────────────────────────────────────────────────────────

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func() *fakeAuthUseCase
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			body: validRegisterRequest(),
			mockSetup: func() *fakeAuthUseCase {
				return &fakeAuthUseCase{
					registerFunc: func(in dto.RegisterRequest) (*dto.UserResponse, error) {
						assert.Equal(t, "user@example.com", in.Email)
						assert.Equal(t, "company-123", in.CompanyID)
						return validUserResponse(), nil
					},
				}
			},
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.UserResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.Equal(t, "user-123", out.ID)
				assert.Equal(t, "user@example.com", out.Email)
				assert.Equal(t, "admin", out.Role)
			},
		},
		{
			name:           "InvalidBody",
			body:           "not valid json",
			mockSetup:      func() *fakeAuthUseCase { return &fakeAuthUseCase{} },
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INVALID_BODY", errResp.Code)
			},
		},
		{
			name: "Validation_EmailPasswordCompanyIDRequired",
			body: dto.RegisterRequest{Email: "", Password: "", CompanyID: ""},
			mockSetup:      func() *fakeAuthUseCase { return &fakeAuthUseCase{} },
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "VALIDATION", errResp.Code)
				assert.Contains(t, errResp.Message, "email")
			},
		},
		{
			name: "Validation_PasswordMinLength",
			body: dto.RegisterRequest{
				Email:     "user@example.com",
				Password:  "short",
				CompanyID: "company-123",
			},
			mockSetup:      func() *fakeAuthUseCase { return &fakeAuthUseCase{} },
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "VALIDATION", errResp.Code)
				assert.Contains(t, errResp.Message, "8")
			},
		},
		{
			name: "Conflict_EmailAlreadyExists",
			body: validRegisterRequest(),
			mockSetup: func() *fakeAuthUseCase {
				return &fakeAuthUseCase{
					registerFunc: func(_ dto.RegisterRequest) (*dto.UserResponse, error) {
						return nil, domain.ErrEmailAlreadyExists
					},
				}
			},
			expectedStatus: http.StatusConflict,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "EMAIL_EXISTS", errResp.Code)
			},
		},
		{
			name: "NotFound_CompanyNotExists",
			body: validRegisterRequest(),
			mockSetup: func() *fakeAuthUseCase {
				return &fakeAuthUseCase{
					registerFunc: func(_ dto.RegisterRequest) (*dto.UserResponse, error) {
						return nil, domain.ErrNotFound
					},
				}
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "COMPANY_NOT_FOUND", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			body: validRegisterRequest(),
			mockSetup: func() *fakeAuthUseCase {
				return &fakeAuthUseCase{
					registerFunc: func(_ dto.RegisterRequest) (*dto.UserResponse, error) {
						return nil, errors.New("db error")
					},
				}
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INTERNAL", errResp.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeUC := tt.mockSetup()
			handler := NewAuthHandler(fakeUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Post("/auth/register", handler.Register)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			if tt.validateBody != nil {
				tt.validateBody(t, resp)
			}
		})
	}
}

// ── Tests Login ─────────────────────────────────────────────────────────────────

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func() *fakeAuthUseCase
		expectedStatus int
		validateBody   func(*testing.T, *http.Response)
	}{
		{
			name: "Success",
			body: validLoginRequest(),
			mockSetup: func() *fakeAuthUseCase {
				return &fakeAuthUseCase{
					loginFunc: func(in dto.LoginRequest) (*dto.LoginResponse, error) {
						assert.Equal(t, "user@example.com", in.Email)
						return validLoginResponse(), nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *http.Response) {
				var out dto.LoginResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
				assert.NotEmpty(t, out.Token)
				assert.Equal(t, "user-123", out.User.ID)
				assert.Equal(t, "user@example.com", out.User.Email)
			},
		},
		{
			name:           "InvalidBody",
			body:           "not valid json",
			mockSetup:      func() *fakeAuthUseCase { return &fakeAuthUseCase{} },
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INVALID_BODY", errResp.Code)
			},
		},
		{
			name: "Validation_EmailPasswordRequired",
			body: dto.LoginRequest{Email: "", Password: ""},
			mockSetup:      func() *fakeAuthUseCase { return &fakeAuthUseCase{} },
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "VALIDATION", errResp.Code)
				assert.Contains(t, errResp.Message, "email")
			},
		},
		{
			name: "Unauthorized_InvalidCredentials_ErrUserNotFound",
			body: validLoginRequest(),
			mockSetup: func() *fakeAuthUseCase {
				return &fakeAuthUseCase{
					loginFunc: func(_ dto.LoginRequest) (*dto.LoginResponse, error) {
						return nil, domain.ErrUserNotFound
					},
				}
			},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "UNAUTHORIZED", errResp.Code)
				assert.Contains(t, errResp.Message, "credenciales")
			},
		},
		{
			name: "Unauthorized_InvalidCredentials_ErrUnauthorized",
			body: validLoginRequest(),
			mockSetup: func() *fakeAuthUseCase {
				return &fakeAuthUseCase{
					loginFunc: func(_ dto.LoginRequest) (*dto.LoginResponse, error) {
						return nil, domain.ErrUnauthorized
					},
				}
			},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "UNAUTHORIZED", errResp.Code)
			},
		},
		{
			name: "Forbidden_AccountInactive",
			body: validLoginRequest(),
			mockSetup: func() *fakeAuthUseCase {
				return &fakeAuthUseCase{
					loginFunc: func(_ dto.LoginRequest) (*dto.LoginResponse, error) {
						return nil, domain.ErrForbidden
					},
				}
			},
			expectedStatus: http.StatusForbidden,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "FORBIDDEN", errResp.Code)
			},
		},
		{
			name: "UseCase_InternalError",
			body: validLoginRequest(),
			mockSetup: func() *fakeAuthUseCase {
				return &fakeAuthUseCase{
					loginFunc: func(_ dto.LoginRequest) (*dto.LoginResponse, error) {
						return nil, errors.New("db error")
					},
				}
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, resp *http.Response) {
				var errResp dto.ErrorResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
				assert.Equal(t, "INTERNAL", errResp.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeUC := tt.mockSetup()
			handler := NewAuthHandler(fakeUC)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Post("/auth/login", handler.Login)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			if tt.validateBody != nil {
				tt.validateBody(t, resp)
			}
		})
	}
}
