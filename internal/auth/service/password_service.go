package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordService определяет интерфейс для работы с паролями
type PasswordService interface {
	HashPassword(password string) (hash, salt string, err error)
	CheckPassword(password, hash, salt string) error
}

// passwordServiceImpl реализует интерфейс PasswordService
type passwordServiceImpl struct{}

// NewPasswordService создаёт новый экземпляр сервиса
func NewPasswordService() PasswordService {
	return &passwordServiceImpl{}
}

// generateSalt генерирует криптографически случайную соль
func (s *passwordServiceImpl) generateSalt() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// HashPassword хеширует пароль с уникальной солью используя bcrypt
// Соль генерируется заново при каждом вызове HashPassword.
// Это гарантирует, что даже одинаковые пароли разных пользователей будут иметь разные хеши в БД.
func (s *passwordServiceImpl) HashPassword(password string) (string, string, error) {
	// Генерируем уникальную соль для каждого пароля
	salt, err := s.generateSalt()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Добавляем соль к паролю перед хешированием
	// Это гарантирует, что одинаковые пароли будут иметь разные хеши
	hash, err := bcrypt.GenerateFromPassword([]byte(password+salt), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), salt, nil
}

// CheckPassword проверяет, соответствует ли пароль хешу
func (s *passwordServiceImpl) CheckPassword(password, hash, salt string) error {
	// Добавляем ту же соль к паролю при проверке
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password+salt))
	if err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}
	return nil
}
