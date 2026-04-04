package myjwt

import (
	"time"

	"github.com/milabo0718/offer-pilot/backend/config"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey      []byte
	expireDuration time.Duration
	issuer         string
	subject        string
}

func NewJWTManager(conf *config.JwtConfig) *JWTManager {
	return &JWTManager{
		secretKey:      []byte(conf.Key),
		expireDuration: time.Duration(conf.ExpireDuration) * time.Hour,
		issuer:         conf.Issuer,
		subject:        conf.Subject,
	}
}

func (m *JWTManager) GenerateToken(id int64, username string) (string, error) {
	claims := Claims{
		ID:       id,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.expireDuration)),
			Issuer:    m.issuer,
			Subject:   m.subject,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *JWTManager) ParseToken(token string) (string, bool) {
	claims := new(Claims)
	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return m.secretKey, nil
	})
	if err != nil || !t.Valid {
		return "", false
	}
	return claims.Username, true
}
