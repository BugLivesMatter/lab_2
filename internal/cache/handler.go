package cache

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StatusHandler возвращает JSON со состоянием Redis и метриками кеша.
// Маршрут: GET /health/redis
// Опционально: ?RedisStatus=1 — то же тело (удобно для явного запроса в документации).
//
// @Summary Статус Redis и метрики кеша
// @Description PING, INFO, DBSIZE, счётчики операций кеша, время последнего успешного доступа приложения к Redis.
// @Tags health
// @Param RedisStatus query string false "Произвольное значение (например 1) для явного запроса статуса"
// @Produce json
// @Success 200 {object} RedisStatusResponse
// @Success 503 {object} RedisStatusResponse "Redis недоступен (PING не прошёл), при этом клиент был сконфигурирован"
// @Router /health/redis [get]
func StatusHandler(svc Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Параметр ?RedisStatus=1 зарезервирован для явного запроса статуса (документация / мониторинг).
		_ = c.Query("RedisStatus")

		resp := svc.RedisStatus(c.Request.Context())

		status := http.StatusOK
		if resp.ClientConfigured && !resp.Connected {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, resp)
	}
}
