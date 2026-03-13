package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User представляет сущность пользователя в системе
// Теги структуры позволяют одной строчкой кода настроить и работу с БД, и формат JSON, без дублирования кода
type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	Phone        string         `gorm:"uniqueIndex" json:"phone,omitempty"`
	PasswordHash string         `gorm:"not null" json:"-"` // Не возвращать в ответах
	Salt         string         `gorm:"not null" json:"-"` // Не возвращать в ответах
	YandexID     string         `gorm:"uniqueIndex" json:"-"`
	VKID         string         `gorm:"uniqueIndex" json:"-"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"` // Soft Delete
}

// TableName указывает имя таблицы в БД
func (User) TableName() string {
	return "users"
}

// BeforeCreate генерирует UUID, если он не задан
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// ToResponse возвращает безопасную версию пользователя для ответов API
// Исключает чувствительные поля: пароль, соль, ID провайдеров
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Phone:     u.Phone,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// UserResponse — DTO для возврата данных пользователя клиенту
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
