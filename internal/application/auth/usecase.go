package auth

import (
	"time"

	"github.com/google/uuid"
	"github.com/tu-usuario/inventory-pro/internal/application/dto"
	"github.com/tu-usuario/inventory-pro/internal/domain"
	"github.com/tu-usuario/inventory-pro/internal/domain/entity"
	"github.com/tu-usuario/inventory-pro/internal/domain/repository"
	"github.com/tu-usuario/inventory-pro/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

// JWTConfig configuración para generación de tokens.
type JWTConfig struct {
	Secret     string
	ExpMinutes int
	Issuer     string
}

// AuthUseCase casos de uso de autenticación: registro y login.
type AuthUseCase struct {
	userRepo    repository.UserRepository
	companyRepo repository.CompanyRepository
	jwtCfg      JWTConfig
}

// NewAuthUseCase construye el caso de uso de auth.
func NewAuthUseCase(userRepo repository.UserRepository, companyRepo repository.CompanyRepository, jwtCfg JWTConfig) *AuthUseCase {
	return &AuthUseCase{userRepo: userRepo, companyRepo: companyRepo, jwtCfg: jwtCfg}
}

// RegisterUser crea un usuario: hashea password con bcrypt y persiste. Devuelve ErrEmailAlreadyExists si el email ya existe en esa company.
func (uc *AuthUseCase) RegisterUser(in dto.RegisterRequest) (*dto.UserResponse, error) {
	existing, _ := uc.userRepo.GetByEmailAndCompany(in.Email, in.CompanyID)
	if existing != nil {
		return nil, domain.ErrEmailAlreadyExists
	}
	company, err := uc.companyRepo.GetByID(in.CompanyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.ErrNotFound // empresa no existe
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	name := in.Name
	if name == "" {
		name = in.Email
	}
	role := in.Role
	if role == "" {
		role = entity.RoleVendedor
	}
	user := &entity.User{
		ID:           uuid.New().String(),
		CompanyID:    in.CompanyID,
		Email:        in.Email,
		PasswordHash: string(hash),
		Name:         name,
		Role:         role,
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := uc.userRepo.Create(user); err != nil {
		return nil, err
	}
	return toUserResponse(user), nil
}

// Login verifica email/password, genera JWT y retorna token + usuario.
func (uc *AuthUseCase) Login(in dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := uc.userRepo.FindByEmail(in.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, domain.ErrUnauthorized
	}
	if user.Status != "active" {
		return nil, domain.ErrForbidden
	}
	token, err := jwt.Generate(uc.jwtCfg.Secret, user.ID, user.CompanyID, user.Role, uc.jwtCfg.Issuer, uc.jwtCfg.ExpMinutes)
	if err != nil {
		return nil, err
	}
	return &dto.LoginResponse{
		Token: token,
		User:  *toUserResponse(user),
	}, nil
}

func toUserResponse(u *entity.User) *dto.UserResponse {
	if u == nil {
		return nil
	}
	return &dto.UserResponse{
		ID:        u.ID,
		CompanyID: u.CompanyID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
