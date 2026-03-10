package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims incluye los claims estándar JWT más los campos propios de la aplicación.
// Se añaden Roles para que el middleware RBAC pueda tomar decisiones sin consultar la DB.
type Claims struct {
	jwt.RegisteredClaims
	UserID    string   `json:"user_id"`
	CompanyID string   `json:"company_id"`
	Roles     []string `json:"roles,omitempty"` // lista de roles
	// Role legacy se mantiene para tokens antiguos / compatibilidad.
	Role string `json:"role,omitempty"`
}

// Generate genera un token JWT firmado que incluye userID, companyID y roles.
func Generate(secret, userID, companyID string, roles []string, issuer string, expMinutes int) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("jwt: secret vacío")
	}
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expMinutes) * time.Minute)),
		},
		UserID:    userID,
		CompanyID: companyID,
		Roles:     roles,
	}
	// Para compatibilidad con código antiguo que lea solo "role", si hay un único rol lo exponemos también.
	if len(roles) == 1 {
		claims.Role = roles[0]
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// Parse valida el token y devuelve userID, companyID y lista de roles.
// Retorna error si el token es inválido, expirado o tiene firma incorrecta.
func Parse(secret, tokenString string) (userID, companyID string, roles []string, err error) {
	if secret == "" {
		return "", "", nil, fmt.Errorf("jwt: secret vacío")
	}
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", "", nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", "", nil, fmt.Errorf("claims inválidos")
	}
	rs := claims.Roles
	// Compatibilidad con tokens antiguos que solo tenían "role".
	if len(rs) == 0 && claims.Role != "" {
		rs = []string{claims.Role}
	}
	return claims.UserID, claims.CompanyID, rs, nil
}
