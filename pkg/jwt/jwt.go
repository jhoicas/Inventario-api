package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims claims estándar más UserID y CompanyID.
type Claims struct {
	jwt.RegisteredClaims
	UserID    string `json:"user_id"`
	CompanyID string `json:"company_id"`
}

// Generate genera un token JWT firmado con los claims indicados.
func Generate(secret string, userID, companyID, issuer string, expMinutes int) (string, error) {
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
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// Parse valida el token y devuelve UserID y CompanyID.
func Parse(secret, tokenString string) (userID, companyID string, err error) {
	if secret == "" {
		return "", "", fmt.Errorf("jwt: secret vacío")
	}
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", "", err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", "", fmt.Errorf("claims inválidos")
	}
	return claims.UserID, claims.CompanyID, nil
}
