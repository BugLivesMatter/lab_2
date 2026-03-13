package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken хранит информацию о сессиях пользователя
// Токен хранится в БД в хешированном виде для безопасности
type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"not null;uniqueIndex"` // Хеш токена, не сам токен!
	ExpiresAt time.Time `gorm:"not null"`
	Revoked   bool      `gorm:"default:false"`
	CreatedAt time.Time
}

// TableName указывает имя таблицы в БД
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsExpired проверяет, истёк ли токен
func (t *RefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsActive возвращает true, если токен действителен и не отозван
func (t *RefreshToken) IsActive() bool {
	return !t.Revoked && !t.IsExpired()
}
