package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/lab2/rest-api/internal/auth/domain"
	"github.com/lab2/rest-api/internal/auth/dto"
	"github.com/lab2/rest-api/internal/auth/repository"
)

// AuthService определяет интерфейс для бизнес-логики авторизации
type AuthService interface {
	Register(ctx context.Context, req *dto.RegisterRequest) (*domain.User, error)
	Login(ctx context.Context, email, password string) (*dto.TokensResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*dto.TokensResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.UserResponse, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
}

// authServiceImpl реализует интерфейс AuthService
type authServiceImpl struct {
	userRepo       repository.UserRepository
	tokenRepo      repository.TokenRepository
	passSvc        PasswordService
	jwtSvc         JWTService
	resetTokenRepo repository.PasswordResetRepository
}

// NewAuthService создаёт новый экземпляр сервиса авторизации
func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	passSvc PasswordService,
	jwtSvc JWTService,
	resetTokenRepo repository.PasswordResetRepository,
) AuthService {
	return &authServiceImpl{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		passSvc:        passSvc,
		jwtSvc:         jwtSvc,
		resetTokenRepo: resetTokenRepo,
	}
}

// Register регистрирует нового пользователя
func (s *authServiceImpl) Register(ctx context.Context, req *dto.RegisterRequest) (*domain.User, error) {
	// Проверка валидности данных
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Проверка, что пользователь с таким email ещё не существует
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, errors.New("пользователь с таким email уже существует")
	}

	// Хеширование пароля с уникальной солью
	passwordHash, salt, err := s.passSvc.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Создание нового пользователя
	user := &domain.User{
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: passwordHash,
		Salt:         salt,
	}

	// Сохранение в БД
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login выполняет вход пользователя и возвращает пару токенов
func (s *authServiceImpl) Login(ctx context.Context, email, password string) (*dto.TokensResponse, error) {
	// Поиск пользователя по email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("неверный email или пароль")
	}

	// Проверка пароля
	if err := s.passSvc.CheckPassword(password, user.PasswordHash, user.Salt); err != nil {
		return nil, errors.New("неверный email или пароль")
	}

	// Генерация JWT токенов
	accessToken, accessExpiry, err := s.jwtSvc.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshExpiry, err := s.jwtSvc.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Хеширование токенов для хранения в БД
	refreshTokenHash := hashToken(refreshToken)
	accessTokenHash := hashToken(accessToken)

	// Сохранение refresh токена в БД (с привязкой access token hash)
	token := &domain.RefreshToken{
		ID:              uuid.New(),
		UserID:          user.ID,
		TokenHash:       refreshTokenHash,
		AccessTokenHash: accessTokenHash,
		ExpiresAt:       time.Now().Add(refreshExpiry),
		Revoked:         false,
	}

	if err := s.tokenRepo.Create(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	// ВОЗВРАЩАЕМ ТОКЕНЫ!
	return &dto.TokensResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresIn:  accessExpiry,
		RefreshExpiresIn: refreshExpiry,
	}, nil
}

// Refresh обновляет пару токенов по refresh токену
func (s *authServiceImpl) Refresh(ctx context.Context, refreshToken string) (*dto.TokensResponse, error) {
	// Валидация refresh токена
	claims, err := s.jwtSvc.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New("невалидный refresh токен")
	}

	// Хеширование токена для поиска в БД
	tokenHash := hashToken(refreshToken)

	// Поиск токена в БД
	storedToken, err := s.tokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	if storedToken == nil {
		return nil, errors.New("токен не найден")
	}

	// Проверка, что токен не отозван и не истёк
	if !storedToken.IsActive() {
		return nil, errors.New("токен отозван или истёк")
	}

	// Отзыв старого токена
	if err := s.tokenRepo.Revoke(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("failed to revoke old token: %w", err)
	}

	// Генерация новой пары токенов
	accessToken, accessExpiry, err := s.jwtSvc.GenerateAccessToken(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, refreshExpiry, err := s.jwtSvc.GenerateRefreshToken(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Сохранение нового refresh токена (с привязкой нового access token hash)
	newTokenHash := hashToken(newRefreshToken)
	newAccessTokenHash := hashToken(accessToken)
	newToken := &domain.RefreshToken{
		ID:              uuid.New(),
		UserID:          claims.UserID,
		TokenHash:       newTokenHash,
		AccessTokenHash: newAccessTokenHash,
		ExpiresAt:       time.Now().Add(refreshExpiry),
		Revoked:         false,
	}

	if err := s.tokenRepo.Create(ctx, newToken); err != nil {
		return nil, fmt.Errorf("failed to save new refresh token: %w", err)
	}

	return &dto.TokensResponse{
		AccessToken:      accessToken,
		RefreshToken:     newRefreshToken,
		AccessExpiresIn:  accessExpiry,
		RefreshExpiresIn: refreshExpiry,
	}, nil
}

// Logout завершает текущую сессию (отзывает один токен)
func (s *authServiceImpl) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	return s.tokenRepo.Revoke(ctx, tokenHash)
}

// LogoutAll завершает все сессии пользователя
func (s *authServiceImpl) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.tokenRepo.RevokeAll(ctx, userID)
}

// GetUserByID возвращает данные пользователя по ID (без чувствительных полей)
func (s *authServiceImpl) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("пользователь не найден")
	}

	// ToResponse() возвращает *UserResponse — просто возвращаем
	return user.ToResponse(), nil
}

// ForgotPassword генерирует токен сброса пароля и отправляет его на email
func (s *authServiceImpl) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		// Не показываем, что пользователь не найден (безопасность)
		return nil
	}

	// Генерируем токен сброса
	resetToken := uuid.New().String()
	expiresAt := time.Now().Add(1 * time.Hour)

	// Сохраняем токен в БД
	token := &domain.PasswordResetToken{
		UserID:    user.ID,
		Token:     resetToken,
		ExpiresAt: expiresAt,
		Used:      false,
	}

	if err := s.resetTokenRepo.Create(ctx, token); err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}

	// В продакшене здесь была бы отправка email
	// Для разработки логируем токен
	// TODO: интегрировать email сервис
	log.Printf("🔑 Reset token for %s: %s", email, resetToken)

	return nil
}

// ResetPassword устанавливает новый пароль по токену
func (s *authServiceImpl) ResetPassword(ctx context.Context, token, newPassword string) error {
	// Проверяем токен
	resetToken, err := s.resetTokenRepo.GetByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to get reset token: %w", err)
	}
	if resetToken == nil {
		return errors.New("невалидный или истёкший токен сброса пароля")
	}

	// Получаем пользователя
	user, err := s.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return errors.New("пользователь не найден")
	}

	// Хэшируем новый пароль
	passwordHash, salt, err := s.passSvc.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Обновляем пароль через интерфейс (без type assertion!)
	if err := s.userRepo.UpdatePassword(ctx, user.ID, passwordHash, salt); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Помечаем токен как использованный
	if err := s.resetTokenRepo.MarkAsUsed(ctx, token); err != nil {
		log.Printf("Warning: failed to mark reset token as used: %v", err)
	}

	// Отзываем все сессии пользователя
	if err := s.tokenRepo.RevokeAll(ctx, user.ID); err != nil {
		log.Printf("Warning: failed to revoke all sessions: %v", err)
	}

	return nil
}
