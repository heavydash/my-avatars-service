package service

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewJWTService(secret string, ttl time.Duration, issuer string) *JWTService {
	return &JWTService{
		secret: []byte(secret),
		ttl:    ttl,
		issuer: issuer,
	}
}

func (s *JWTService) GenerateToken(userID string) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTService) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrTokenInvalidId
}
