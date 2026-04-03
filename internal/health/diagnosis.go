package health

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/lab2/rest-api/internal/cache"
	"github.com/lab2/rest-api/internal/category/domain"
	categoryrepo "github.com/lab2/rest-api/internal/category/repository"
	"github.com/lab2/rest-api/pkg/pagination"
)

// categoriesListCachePayload — тот же JSON, что кладёт categoryService.List в Redis (см. internal/category/service/service.go).
type categoriesListCachePayload struct {
	Categories []domain.Category `json:"categories"`
	Total      int64             `json:"total"`
	TotalPages int               `json:"total_pages"`
}

// DiagnosisResponse — сравнение латентности PostgreSQL и Redis и оценка выгоды от кеша.
type DiagnosisResponse struct {
	CheckedAt string `json:"checkedAt"`

	PostgreSQL DiagnosisPostgresSection `json:"postgresql"`
	Redis      DiagnosisRedisSection    `json:"redis"`

	// ComparisonSimple — SELECT 1 vs PING.
	ComparisonSimple *DiagnosisComparison `json:"comparisonSimple,omitempty"`
	// ComparisonWorkload — repository.List (как при промахе кеша) vs cache.Get (как при попадании).
	ComparisonWorkload *DiagnosisComparison `json:"comparisonWorkload,omitempty"`
	// ComparisonMissVsHit — полный промах (List + cache.Set) vs cache hit (Get), как два последовательных запроса GET /categories.
	ComparisonMissVsHit *DiagnosisComparison `json:"comparisonMissVsHit,omitempty"`

	Notes string `json:"notes,omitempty"`
}

// DiagnosisPostgresSection — замеры обращений к PostgreSQL.
type DiagnosisPostgresSection struct {
	OK bool `json:"ok"`

	SimpleQuery string  `json:"simpleQuery,omitempty"`
	SimpleMs    float64 `json:"simpleLatencyMs,omitempty"`
	SimpleError string  `json:"simpleError,omitempty"`

	// Ниже — тот же путь данных, что GET /categories при промахе кеша.
	ListRepoMethod string `json:"listRepositoryMethod,omitempty"`
	Page           int    `json:"page"`
	Limit          int    `json:"limit"`
	CacheKey       string `json:"cacheKey,omitempty"`

	// ListMs — только CategoryRepository.List (Count + Find внутри репозитория).
	ListMs float64 `json:"listLatencyMs,omitempty"`
	// ListRowCount — число записей на странице.
	ListRowCount int `json:"listRowCount,omitempty"`
	Total        int64 `json:"total,omitempty"`
	ListError    string `json:"listError,omitempty"`

	// ApproxPayloadBytes — размер JSON тела, как у cache.Service.Set (для справки).
	ApproxPayloadBytes int `json:"approxPayloadJsonBytes,omitempty"`
}

// DiagnosisRedisSection — замеры Redis / cache.Service.
type DiagnosisRedisSection struct {
	OK bool `json:"ok"`

	PingMs    float64 `json:"pingLatencyMs,omitempty"`
	PingError string  `json:"pingError,omitempty"`

	// CacheSetMs / CacheGetMs — те же вызовы, что в categoryService.List.
	CacheSetMs float64 `json:"cacheSetLatencyMs,omitempty"`
	CacheGetMs float64 `json:"cacheGetLatencyMs,omitempty"`
	CacheHit   bool    `json:"cacheGetHit,omitempty"`

	CacheError          string `json:"cacheError,omitempty"`
	ClientNotConfigured bool   `json:"clientNotConfigured,omitempty"`
}

// DiagnosisComparison — сравнение двух латентностей в миллисекундах.
type DiagnosisComparison struct {
	PostgresMs float64 `json:"postgresqlMs"`
	RedisMs    float64 `json:"redisMs"`
	// RedisFasterPercent — доля сокращения времени относительно PostgreSQL: (pg-redis)/pg*100 при pg>redis.
	// Это не «во сколько раз быстрее»; множитель см. RedisSpeedupFactor (= pg/redis).
	RedisFasterPercent *float64 `json:"redisFasterThanPostgresPercent,omitempty"`
	RedisSpeedupFactor *float64 `json:"redisSpeedupFactor,omitempty"`
	Summary            string   `json:"summary"`
}

func msSince(t time.Time) float64 {
	return float64(time.Since(t).Microseconds()) / 1000.0
}

func buildComparison(pgMs, redisMs float64, label string) *DiagnosisComparison {
	if pgMs < 0 || redisMs < 0 {
		return nil
	}
	c := &DiagnosisComparison{
		PostgresMs: pgMs,
		RedisMs:    redisMs,
	}
	if pgMs > 0 && redisMs < pgMs {
		p := (pgMs - redisMs) / pgMs * 100
		c.RedisFasterPercent = &p
	}
	if redisMs > 0 {
		f := pgMs / redisMs
		c.RedisSpeedupFactor = &f
	}
	if c.RedisFasterPercent != nil && c.RedisSpeedupFactor != nil {
		// Процент — сокращение латентности относительно времени БД; «во сколько раз» — по RedisSpeedupFactor.
		c.Summary = fmt.Sprintf(
			"%s: %.3f мс (Redis) против %.3f мс (PostgreSQL) — кеш примерно в %.2f раз быстрее по времени; латентность Redis короче на %.1f%% от времени БД",
			label, redisMs, pgMs, *c.RedisSpeedupFactor, *c.RedisFasterPercent,
		)
	} else if c.RedisFasterPercent != nil {
		c.Summary = fmt.Sprintf("%s: %.3f мс (Redis) против %.3f мс (PostgreSQL); латентность Redis короче на %.1f%% от времени БД", label, redisMs, pgMs, *c.RedisFasterPercent)
	} else {
		c.Summary = fmt.Sprintf("%s: при данных замерах кеш не быстрее (pg=%.3f мс, redis=%.3f мс)", label, pgMs, redisMs)
	}
	return c
}

func normalizePageLimit(page, limit int) (int, int, int) {
	if page < 1 {
		page = pagination.DefaultPage
	}
	if limit < 1 {
		limit = pagination.DefaultLimit
	}
	if limit > pagination.MaxLimit {
		limit = pagination.MaxLimit
	}
	offset := (page - 1) * limit
	return page, limit, offset
}

// RunDiagnosisParams — параметры прогона (те же page/limit, что у GET /categories).
type RunDiagnosisParams struct {
	Page int
	Limit int
}

// RunDiagnosis выполняет замеры. Перед записью в кеш удаляется ключ страницы — как холодный промах для этой пары page/limit.
func RunDiagnosis(ctx context.Context, db *gorm.DB, rdb *redis.Client, repo categoryrepo.CategoryRepository, cacheSvc cache.Service, cacheTTL time.Duration, p RunDiagnosisParams) DiagnosisResponse {
	out := DiagnosisResponse{
		CheckedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}

	page, limit, offset := normalizePageLimit(p.Page, p.Limit)
	out.PostgreSQL.Page = page
	out.PostgreSQL.Limit = limit
	cacheKey := cache.CategoriesListKey(page, limit)
	out.PostgreSQL.CacheKey = cacheKey
	out.PostgreSQL.ListRepoMethod = "CategoryRepository.List(ctx, offset, limit) — тот же вызов, что при промахе кеша в CategoryService.List"

	// --- PostgreSQL: SELECT 1 ---
	t0 := time.Now()
	var one int
	err := db.WithContext(ctx).Raw("SELECT 1").Scan(&one).Error
	out.PostgreSQL.SimpleMs = msSince(t0)
	out.PostgreSQL.SimpleQuery = "SELECT 1"
	if err != nil {
		out.PostgreSQL.SimpleError = err.Error()
	} else {
		out.PostgreSQL.OK = true
	}

	// --- Redis PING (сырой клиент, как в cache.NewRedisClient) ---
	if rdb == nil {
		out.Redis.ClientNotConfigured = true
		out.Redis.PingError = "клиент Redis не инициализирован"
	} else {
		tPing := time.Now()
		if pErr := rdb.Ping(ctx).Err(); pErr != nil {
			out.Redis.PingError = pErr.Error()
		} else {
			out.Redis.OK = true
			out.Redis.PingMs = msSince(tPing)
		}
	}

	// Сброс кеша страницы — воспроизводим промах без обхода сервиса.
	_ = cacheSvc.Del(ctx, cacheKey)
	out.Notes = fmt.Sprintf("Перед замером выполнен cache.Del(%q) — для этой пары page/limit следующий GET /categories получит промах кеша.", cacheKey)

	// --- Тот же путь, что CategoryService.List при промахе ---
	tList := time.Now()
	categories, total, listErr := repo.List(ctx, offset, limit)
	out.PostgreSQL.ListMs = msSince(tList)
	if listErr != nil {
		out.PostgreSQL.ListError = listErr.Error()
		out.Redis.CacheError = "пропущено: ошибка List"
		return out
	}
	out.PostgreSQL.ListRowCount = len(categories)
	out.PostgreSQL.Total = total

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	payload := categoriesListCachePayload{
		Categories: categories,
		Total:      total,
		TotalPages: totalPages,
	}
	if b, jErr := json.Marshal(payload); jErr == nil {
		out.PostgreSQL.ApproxPayloadBytes = len(b)
	}

	tSet := time.Now()
	if sErr := cacheSvc.Set(ctx, cacheKey, payload, cacheTTL); sErr != nil {
		out.Redis.CacheError = "cache.Set: " + sErr.Error()
		return out
	}
	out.Redis.CacheSetMs = msSince(tSet)

	var warm categoriesListCachePayload
	tGet := time.Now()
	hit, gErr := cacheSvc.Get(ctx, cacheKey, &warm)
	out.Redis.CacheGetMs = msSince(tGet)
	if gErr != nil {
		out.Redis.CacheError = "cache.Get: " + gErr.Error()
		return out
	}
	out.Redis.CacheHit = hit

	if out.PostgreSQL.SimpleError == "" && out.Redis.PingError == "" && rdb != nil {
		out.ComparisonSimple = buildComparison(out.PostgreSQL.SimpleMs, out.Redis.PingMs, "Минимальный round-trip (SELECT 1 vs PING)")
	}

	// Ядро: стоимость чтения из БД vs чтения из кеша тем же cache.Service.Get.
	out.ComparisonWorkload = buildComparison(out.PostgreSQL.ListMs, out.Redis.CacheGetMs, "Список категорий: repository.List vs cache hit (cache.Service.Get)")

	missTotal := out.PostgreSQL.ListMs + out.Redis.CacheSetMs
	out.ComparisonMissVsHit = buildComparison(missTotal, out.Redis.CacheGetMs, "Полный промах кеша (List+Set) vs cache hit (Get), как два запроса GET /categories")

	return out
}
