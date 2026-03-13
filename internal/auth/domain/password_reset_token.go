package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PasswordResetToken представляет токен сброса пароля
type PasswordResetToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"userId"`
	Token     string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"token"`
	ExpiresAt time.Time `gorm:"type:timestamptz;not null" json:"expiresAt"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `gorm:"type:timestamptz;default:now()" json:"createdAt"`
}

// TableName указывает имя таблицы в БД
func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}

// BeforeCreate генерирует UUID, если он не задан
func (t *PasswordResetToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// IsExpired проверяет, истёк ли токен
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}
