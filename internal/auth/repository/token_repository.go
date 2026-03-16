package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/lab2/rest-api/internal/auth/domain"
)

// TokenRepository определяет интерфейс для работы с refresh-токенами в БД
type TokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	GetByAccessTokenHash(ctx context.Context, accessTokenHash string) (*domain.RefreshToken, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAll(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

// tokenRepositoryImpl реализует интерфейс TokenRepository
type tokenRepositoryImpl struct {
	db *gorm.DB
}

// NewTokenRepository создаёт новый экземпляр репозитория
func NewTokenRepository(db *gorm.DB) TokenRepository {
	return &tokenRepositoryImpl{db: db}
}

// Create создаёт новый refresh-токен в БД
func (r *tokenRepositoryImpl) Create(ctx context.Context, token *domain.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// GetByHash находит токен по хешу
func (r *tokenRepositoryImpl) GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	var token domain.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

// GetByAccessTokenHash ищет активную сессию по хэшу access token.
// Возвращает nil, если сессия не найдена или уже отозвана.
func (r *tokenRepositoryImpl) GetByAccessTokenHash(ctx context.Context, accessTokenHash string) (*domain.RefreshToken, error) {
	var token domain.RefreshToken
	err := r.db.WithContext(ctx).
		Where("access_token_hash = ? AND revoked = ? AND expires_at > ?", accessTokenHash, false, time.Now()).
		First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

// GetActiveByUserID возвращает все активные токены пользователя
func (r *tokenRepositoryImpl) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error) {
	var tokens []*domain.RefreshToken
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked = ? AND expires_at > ?", userID, false, time.Now()).
		Find(&tokens).Error
	return tokens, err
}

// Revoke помечает токен как отозванный
func (r *tokenRepositoryImpl) Revoke(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Update("revoked", true).Error
}

// RevokeAll отзывает все токены пользователя
func (r *tokenRepositoryImpl) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked = ?", userID, false).
		Update("revoked", true).Error
}

// DeleteExpired удаляет истёкшие токены (опционально, для очистки)
func (r *tokenRepositoryImpl) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&domain.RefreshToken{}).Error
}
