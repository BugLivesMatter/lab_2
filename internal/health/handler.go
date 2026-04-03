package health

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/lab2/rest-api/internal/cache"
	categoryrepo "github.com/lab2/rest-api/internal/category/repository"
	"github.com/lab2/rest-api/pkg/pagination"
)

// DiagnosisHandler возвращает сравнение PostgreSQL и Redis на том же пути, что GET /categories.
//
// @Summary Диагностика БД и Redis (как GET /categories)
// @Description Использует CategoryRepository.List и cache.Service с тем же ключом и JSON, что CategoryService.List. Перед замером удаляет кеш страницы (cache.Del). Параметры page и limit — как у GET /categories.
// @Tags health
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Размер страницы (макс. 100)" default(10)
// @Produce json
// @Success 200 {object} DiagnosisResponse
// @Router /health/diagnosis [get]
func DiagnosisHandler(db *gorm.DB, rdb *redis.Client, repo categoryrepo.CategoryRepository, cacheSvc cache.Service, cacheTTL time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		page := pagination.DefaultPage
		if v := c.Query("page"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				page = n
			}
		}
		limit := pagination.DefaultLimit
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}

		resp := RunDiagnosis(ctx, db, rdb, repo, cacheSvc, cacheTTL, RunDiagnosisParams{Page: page, Limit: limit})
		c.JSON(http.StatusOK, resp)
	}
}
