package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/lab2/rest-api/internal/auth/domain"
	"github.com/lab2/rest-api/internal/auth/dto"
	"github.com/lab2/rest-api/internal/auth/repository"
)

// OAuthService определяет интерфейс для OAuth авторизации
type OAuthService interface {
	GetAuthorizationURL(provider string) (string, string, error)
	HandleCallback(ctx context.Context, provider, code, state string) (*domain.User, *dto.TokensResponse, error)
}

// oauthServiceImpl реализует интерфейс OAuthService
type oauthServiceImpl struct {
	userRepo  repository.UserRepository
	tokenRepo repository.TokenRepository
	passSvc   PasswordService
	jwtSvc    JWTService
	config    *OAuthConfig
}

// OAuthConfig содержит конфигурацию OAuth провайдеров
type OAuthConfig struct {
	YandexClientID     string
	YandexClientSecret string
	YandexRedirectURI  string
}

// NewOAuthService создаёт новый экземпляр OAuth сервиса
func NewOAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	passSvc PasswordService,
	jwtSvc JWTService,
	config *OAuthConfig,
) OAuthService {
	return &oauthServiceImpl{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		passSvc:   passSvc,
		jwtSvc:    jwtSvc,
		config:    config,
	}
}

// GetAuthorizationURL возвращает URL для авторизации через провайдера
func (s *oauthServiceImpl) GetAuthorizationURL(provider string) (string, string, error) {
	switch provider {
	case "yandex":
		return s.getYandexAuthURL()
	default:
		return "", "", errors.New("неподдерживаемый провайдер")
	}
}

// getYandexAuthURL генерирует URL для авторизации через Яндекс
func (s *oauthServiceImpl) getYandexAuthURL() (string, string, error) {
	state := generateOAuthState()

	params := url.Values{}
	params.Set("client_id", s.config.YandexClientID)
	params.Set("redirect_uri", s.config.YandexRedirectURI)
	params.Set("response_type", "code")
	params.Set("state", state)

	return "https://oauth.yandex.ru/authorize?" + params.Encode(), state, nil
}

// HandleCallback обрабатывает callback от OAuth провайдера
func (s *oauthServiceImpl) HandleCallback(ctx context.Context, provider, code, state string) (*domain.User, *dto.TokensResponse, error) {
	switch provider {
	case "yandex":
		return s.handleYandexCallback(ctx, code, state)
	default:
		return nil, nil, errors.New("неподдерживаемый провайдер")
	}
}

// handleYandexCallback обрабатывает callback от Яндекса
func (s *oauthServiceImpl) handleYandexCallback(ctx context.Context, code, state string) (*domain.User, *dto.TokensResponse, error) {
	// 1. Обмениваем code на access_token
	tokenResp, err := s.getYandexToken(code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get token: %w", err)
	}

	// 2. Получаем информацию о пользователе
	yandexUser, err := s.getYandexUserInfo(tokenResp.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// 3. Ищем пользователя по Yandex ID
	user, err := s.userRepo.GetByYandexID(ctx, yandexUser.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 4. Если не найден — создаём нового
	if user == nil {
		user = &domain.User{
			Email:    yandexUser.Email,
			YandexID: yandexUser.ID,
			VKID:     "", // Пустая строка, т.к. поле string (не указатель)
		}
		// Генерируем случайный пароль (он не будет использоваться)
		randomPassword := uuid.New().String()
		passwordHash, salt, err := s.passSvc.HashPassword(randomPassword)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = passwordHash
		user.Salt = salt

		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	// 5. Генерируем JWT токены
	accessToken, accessExpiry, err := s.jwtSvc.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshExpiry, err := s.jwtSvc.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// 6. Сохраняем refresh token (с привязкой access token hash)
	refreshTokenHash := hashToken(refreshToken)
	accessTokenHash := hashToken(accessToken)
	token := &domain.RefreshToken{
		ID:              uuid.New(),
		UserID:          user.ID,
		TokenHash:       refreshTokenHash,
		AccessTokenHash: accessTokenHash,
		ExpiresAt:       time.Now().Add(refreshExpiry),
		Revoked:         false,
	}

	if err := s.tokenRepo.Create(ctx, token); err != nil {
		return nil, nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	tokens := &dto.TokensResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresIn:  accessExpiry,
		RefreshExpiresIn: refreshExpiry,
	}

	return user, tokens, nil
}

// getYandexToken обменивает code на access_token
func (s *oauthServiceImpl) getYandexToken(code string) (*YandexTokenResponse, error) {
	resp, err := http.PostForm("https://oauth.yandex.ru/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {s.config.YandexClientID},
		"client_secret": {s.config.YandexClientSecret},
		"redirect_uri":  {s.config.YandexRedirectURI},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp YandexTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// getYandexUserInfo получает информацию о пользователе из Яндекса
func (s *oauthServiceImpl) getYandexUserInfo(accessToken string) (*dto.OAuthUserInfo, error) {
	req, err := http.NewRequest("GET", "https://login.yandex.ru/info", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo dto.OAuthUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// YandexTokenResponse содержит ответ от Яндекс OAuth
type YandexTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// generateOAuthState генерирует state для OAuth flow
func generateOAuthState() string {
	data := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"nonce":     uuid.New().String(),
	}
	jsonData, _ := json.Marshal(data)
	return base64.StdEncoding.EncodeToString(jsonData)
}

// hashToken хеширует токен для безопасного хранения в БД
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
