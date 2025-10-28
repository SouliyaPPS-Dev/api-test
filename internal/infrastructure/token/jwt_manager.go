package token

import (
	"errors"
	"time"

	usecase "backoffice/backend/internal/usecase/auth"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager issues and validates JWT tokens.
type JWTManager struct {
	secret     []byte
	expiration time.Duration
	issuer     string
}

// NewJWTManager constructs a manager with the provided secret and expiration.
func NewJWTManager(secret string, expiration time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		secret:     []byte(secret),
		expiration: expiration,
		issuer:     issuer,
	}
}

// Ensure JWTManager implements the TokenManager interface.
var _ usecase.TokenManager = (*JWTManager)(nil)

// Claims represents token claims.
type Claims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

// Generate creates a signed JWT containing the user id.
func (m *JWTManager) Generate(userID string) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expiration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Validate parses and validates the token returning the user id when valid.
func (m *JWTManager) Validate(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token claims")
	}
	return claims.UserID, nil
}
