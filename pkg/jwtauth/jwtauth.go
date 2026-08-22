// Package jwtauth issues and validates the JWT tokens used for authentication.
// It only knows about claims and signing — it has no concept of users,
// repositories, or HTTP.
package jwtauth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT claim set; user identity travels in these fields.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// TokenExpiry is the documented 24-hour token lifetime.
const TokenExpiry = 24 * time.Hour

// Manager issues and validates tokens against a fixed secret.
type Manager struct {
	secret []byte
}

// New returns a Manager bound to the given signing secret.
func New(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

// Generate produces a signed JWT for the given user identity.
func (m *Manager) Generate(userID, email, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse validates a token string and returns its claims, or an error if the
// token is invalid or expired.
func (m *Manager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
