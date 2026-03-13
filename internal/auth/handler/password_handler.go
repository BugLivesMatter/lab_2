package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lab2/rest-api/internal/auth/dto"
	"github.com/lab2/rest-api/internal/auth/service"
)

// PasswordHandler обрабатывает запросы сброса пароля
type PasswordHandler struct {
	authService service.AuthService
}

// NewPasswordHandler создаёт новый экземпляр хендлера
func NewPasswordHandler(authService service.AuthService) *PasswordHandler {
	return &PasswordHandler{
		authService: authService,
	}
}

// ForgotPassword обрабатывает POST /auth/forgot-password
func (h *PasswordHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные запроса"})
		return
	}

	if err := h.authService.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при обработке запроса"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "если пользователь существует, письмо для сброса пароля отправлено",
	})
}

// ResetPassword обрабатывает POST /auth/reset-password
func (h *PasswordHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные запроса"})
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "пароль успешно изменён",
	})
}
