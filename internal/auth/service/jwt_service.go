package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims представляет claims JWT токена
type Claims struct {
	UserID uuid.UUID `json:"sub"`
	jwt.RegisteredClaims
}

// JWTService определяет интерфейс для работы с JWT токенами
type JWTService interface {
	GenerateAccessToken(userID uuid.UUID) (string, time.Duration, error)
	GenerateRefreshToken(userID uuid.UUID) (string, time.Duration, error)
	ValidateAccessToken(tokenString string) (*Claims, error)
	ValidateRefreshToken(tokenString string) (*Claims, error)
}

// jwtServiceImpl реализует интерфейс JWTService
type jwtServiceImpl struct {
	accessSecret  []byte
	refreshSecret []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewJWTService создаёт новый экземпляр JWT сервиса
func NewJWTService(accessSecret, refreshSecret string, accessExpiry, refreshExpiry time.Duration) JWTService {
	return &jwtServiceImpl{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// GenerateAccessToken генерирует Access токен с коротким временем жизни
func (s *jwtServiceImpl) GenerateAccessToken(userID uuid.UUID) (string, time.Duration, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.accessSecret)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, s.accessExpiry, nil
}

// GenerateRefreshToken генерирует Refresh токен с длительным временем жизни
func (s *jwtServiceImpl) GenerateRefreshToken(userID uuid.UUID) (string, time.Duration, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.refreshSecret)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, s.refreshExpiry, nil
}

// ValidateAccessToken проверяет валидность Access токена
func (s *jwtServiceImpl) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.accessSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid access token claims")
	}

	return claims, nil
}

// ValidateRefreshToken проверяет валидность Refresh токена
func (s *jwtServiceImpl) ValidateRefreshToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.refreshSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid refresh token claims")
	}

	return claims, nil
}
