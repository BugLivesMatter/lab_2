package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lab2/rest-api/internal/auth/service"
)

// AuthMiddleware создаёт middleware для проверки JWT токена
func AuthMiddleware(jwtService service.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Извлекаем токен из HttpOnly cookie
		tokenString, err := c.Cookie("access_token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "токен авторизации не найден",
			})
			return
		}

		// 2. Проверяем токен
		claims, err := jwtService.ValidateAccessToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "невалидный или истёкший токен",
			})
			return
		}

		// 3. Сохраняем userID в контекст для использования в хендлерах
		c.Set("userID", claims.UserID.String())
		c.Set("userEmail", claims.UserID.String()) // Можно добавить email в claims

		// 4. Передаём управление следующему хендлеру
		c.Next()
	}
}

// CORS middleware (опционально, для фронтенда)
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
