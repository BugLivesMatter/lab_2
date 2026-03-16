package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken хранит информацию о сессиях пользователя.
// Оба токена хранятся в виде SHA-256 хэшей для безопасности.
type RefreshToken struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null" json:"userId"`
	TokenHash       string    `gorm:"type:text;not null;uniqueIndex" json:"-"`
	AccessTokenHash string    `gorm:"type:text;uniqueIndex" json:"-"`
	ExpiresAt       time.Time `gorm:"type:timestamptz;not null" json:"expiresAt"`
	Revoked         bool      `gorm:"default:false" json:"revoked"`
	CreatedAt       time.Time `gorm:"type:timestamptz;default:now()" json:"createdAt"`
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
