// Package handler содержит HTTP-обработчики (контроллеры) для модуля авторизации.
// Примечание: Комментарии с аннотациями @Summary, @Tags, @Param и т.д.
// используются для автоматической генерации API-документации через Swagger/OpenAPI.
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/lab2/rest-api/internal/auth/dto"
	"github.com/lab2/rest-api/internal/auth/service"
)

// AuthHandler обрабатывает HTTP запросы для авторизации
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler создаёт новый экземпляр хендлера
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register обрабатывает POST /auth/register
// @Summary Регистрация пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Данные для регистрации"
// @Success 201 {object} map[string]interface{}
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные запроса"})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, errors.New("пользователь с таким email уже существует")) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "пользователь успешно зарегистрирован",
		"userId":  user.ID,
	})
}

// Login обрабатывает POST /auth/login
// @Summary Вход пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Данные для входа"
// @Success 200 {object} map[string]interface{}
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные запроса"})
		return
	}

	tokens, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Установка HttpOnly cookies
	// Access Token (короткое время жизни)
	c.SetCookie(
		"access_token",
		tokens.AccessToken, // Токен устанавливается в authService, здесь только метаданные
		int(tokens.AccessExpiresIn.Seconds()),
		"/",
		"",
		false, // Secure: false для localhost, true для HTTPS
		true,  // HttpOnly: true (JavaScript не имеет доступа)
	)

	// Refresh Token (длительное время жизни)
	c.SetCookie(
		"refresh_token",
		tokens.RefreshToken,
		int(tokens.RefreshExpiresIn.Seconds()),
		"/",
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"message":          "успешный вход",
		"accessExpiresIn":  tokens.AccessExpiresIn.String(),
		"refreshExpiresIn": tokens.RefreshExpiresIn.String(),
	})
}

// Refresh обрабатывает POST /auth/refresh
// @Summary Обновление токенов
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	// Получаем refresh токен из cookies
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh токен не найден"})
		return
	}

	tokens, err := h.authService.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Обновляем cookies с НОВЫМИ токенами
	// Access Token
	c.SetCookie(
		"access_token",
		tokens.AccessToken, // ← Было: "", стало: tokens.AccessToken
		int(tokens.AccessExpiresIn.Seconds()),
		"/",
		"",
		false,
		true,
	)

	// Refresh Token
	c.SetCookie(
		"refresh_token",
		tokens.RefreshToken, // ← Было: "", стало: tokens.RefreshToken
		int(tokens.RefreshExpiresIn.Seconds()),
		"/",
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"message":          "токены обновлены",
		"accessExpiresIn":  tokens.AccessExpiresIn.String(),
		"refreshExpiresIn": tokens.RefreshExpiresIn.String(),
	})
}

// WhoAmI обрабатывает GET /auth/whoami
// @Summary Получение данных текущего пользователя
// @Tags auth
// @Produce json
// @Success 200 {object} domain.UserResponse
// @Router /auth/whoami [get]
func (h *AuthHandler) WhoAmI(c *gin.Context) {
	// UserID добавляется в контекст middleware
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка обработки запроса"})
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// Logout обрабатывает POST /auth/logout
// @Summary Выход из текущей сессии
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh токен не найден"})
		return
	}

	if err := h.authService.Logout(c.Request.Context(), refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при выходе"})
		return
	}

	// Удаляем cookies
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"message": "успешный выход"})
}

// LogoutAll обрабатывает POST /auth/logout-all
// @Summary Выход из всех сессий
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "пользователь не авторизован"})
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка обработки запроса"})
		return
	}

	if err := h.authService.LogoutAll(c.Request.Context(), userUUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при выходе из всех сессий"})
		return
	}

	// Удаляем cookies
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"message": "успешный выход из всех сессий"})
}
