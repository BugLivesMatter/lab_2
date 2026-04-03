package cache

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStatusResponse — полный ответ эндпоинта проверки Redis / кеша.
type RedisStatusResponse struct {
	// IsEnabled — кеш считается включённым: CACHE_ENABLED=true и клиент создан.
	IsEnabled bool `json:"isEnabled"`
	// CacheConfigEnabled — значение CACHE_ENABLED из конфигурации.
	CacheConfigEnabled bool `json:"cacheConfigEnabled"`
	// ClientConfigured — клиент Redis был успешно инициализирован (не nil).
	ClientConfigured bool `json:"clientConfigured"`
	// Connected — успешный PING на момент запроса.
	Connected bool `json:"connected"`
	// PingLatencyMs — задержка PING в миллисекундах.
	PingLatencyMs float64 `json:"pingLatencyMs,omitempty"`
	// CheckedAt — время выполнения проверки (UTC, RFC3339Nano).
	CheckedAt string `json:"checkedAt"`
	// LastSuccessfulAccessAt — последнее успешное обращение приложения к Redis (UTC).
	LastSuccessfulAccessAt *string `json:"lastSuccessfulAccessAt,omitempty"`
	// Usage — агрегированная нагрузка по операциям кеша с момента старта процесса.
	Usage *RedisUsageMetrics `json:"usage,omitempty"`
	// Server — выборка полей из Redis INFO.
	Server *RedisServerInfo `json:"server,omitempty"`
	// KeysInDB — приблизительное число ключей в текущей БД (DBSIZE).
	KeysInDB int64 `json:"keysInDb,omitempty"`
	// Error — текст ошибки PING / INFO / DBSIZE (если была).
	Error string `json:"error,omitempty"`
}

// RedisUsageMetrics — счётчики успешных операций кеша.
type RedisUsageMetrics struct {
	GetHits           uint64  `json:"getRequests"`
	SetWrites         uint64  `json:"setWrites"`
	DelSingle         uint64  `json:"delSingle"`
	DelByPattern      uint64  `json:"delByPattern"`
	ExistsChecks      uint64  `json:"existsChecks"`
	TotalCacheTouches uint64  `json:"totalCacheOperations"`
	HitRatioEstimate  float64 `json:"hitRatioEstimate,omitempty"`
}

// RedisServerInfo — метрики экземпляра Redis (из INFO).
type RedisServerInfo struct {
	RedisVersion           string  `json:"redisVersion"`
	Role                   string  `json:"role"`
	UptimeSeconds          int64   `json:"uptimeSeconds"`
	ConnectedClients       int64   `json:"connectedClients"`
	UsedMemoryBytes        int64   `json:"usedMemoryBytes"`
	UsedMemoryHuman        string  `json:"usedMemoryHuman"`
	UsedMemoryPeakHuman    string  `json:"usedMemoryPeakHuman,omitempty"`
	MaxMemoryBytes         int64   `json:"maxMemoryBytes,omitempty"`
	MemFragmentationRatio  float64 `json:"memFragmentationRatio,omitempty"`
	TotalConnectionsRecv   int64   `json:"totalConnectionsReceived"`
	TotalCommandsProcessed int64   `json:"totalCommandsProcessed"`
	InstantaneousOpsPerSec int64   `json:"instantaneousOpsPerSec,omitempty"`
	KeyspaceHits           int64   `json:"keyspaceHits"`
	KeyspaceMisses         int64   `json:"keyspaceMisses"`
}

func parseRedisInfo(raw string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, ':')
		if idx <= 0 {
			continue
		}
		out[line[:idx]] = strings.TrimSpace(line[idx+1:])
	}
	return out
}

func infoInt64(m map[string]string, key string) int64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func infoFloat64(m map[string]string, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0
	}
	return f
}

func buildServerInfo(m map[string]string) *RedisServerInfo {
	hits := infoInt64(m, "keyspace_hits")
	misses := infoInt64(m, "keyspace_misses")
	return &RedisServerInfo{
		RedisVersion:           m["redis_version"],
		Role:                   m["role"],
		UptimeSeconds:          infoInt64(m, "uptime_in_seconds"),
		ConnectedClients:       infoInt64(m, "connected_clients"),
		UsedMemoryBytes:        infoInt64(m, "used_memory"),
		UsedMemoryHuman:        m["used_memory_human"],
		UsedMemoryPeakHuman:    m["used_memory_peak_human"],
		MaxMemoryBytes:         infoInt64(m, "maxmemory"),
		MemFragmentationRatio:  infoFloat64(m, "mem_fragmentation_ratio"),
		TotalConnectionsRecv:   infoInt64(m, "total_connections_received"),
		TotalCommandsProcessed: infoInt64(m, "total_commands_processed"),
		InstantaneousOpsPerSec: infoInt64(m, "instantaneous_ops_per_sec"),
		KeyspaceHits:           hits,
		KeyspaceMisses:         misses,
	}
}

// collectRedisStatus выполняет PING, INFO, DBSIZE и собирает метрики приложения.
func collectRedisStatus(ctx context.Context, client *redis.Client, configEnabled bool, metrics *accessMetrics) RedisStatusResponse {
	if metrics == nil {
		metrics = newAccessMetrics()
	}
	now := time.Now().UTC()
	resp := RedisStatusResponse{
		CacheConfigEnabled: configEnabled,
		ClientConfigured:   client != nil,
		IsEnabled:          configEnabled && client != nil,
		CheckedAt:          now.Format(time.RFC3339Nano),
	}

	last, hasLast, gets, sets, dels, exists, pDel := metrics.snapshot()
	if hasLast {
		s := last.Format(time.RFC3339Nano)
		resp.LastSuccessfulAccessAt = &s
	}
	total := gets + sets + dels + exists + pDel
	resp.Usage = &RedisUsageMetrics{
		GetHits:           gets,
		SetWrites:         sets,
		DelSingle:         dels,
		DelByPattern:      pDel,
		ExistsChecks:      exists,
		TotalCacheTouches: total,
	}

	if client == nil {
		if !configEnabled {
			resp.Error = "redis: клиент не создан, CACHE_ENABLED=false"
		} else {
			resp.Error = "redis: клиент не создан (ошибка подключения при старте)"
		}
		return resp
	}

	t0 := time.Now()
	if err := client.Ping(ctx).Err(); err != nil {
		resp.Error = err.Error()
		resp.PingLatencyMs = float64(time.Since(t0).Microseconds()) / 1000.0
		return resp
	}
	resp.Connected = true
	resp.PingLatencyMs = float64(time.Since(t0).Microseconds()) / 1000.0

	infoRaw, err := client.Info(ctx).Result()
	if err != nil {
		resp.Error = "info: " + err.Error()
		return resp
	}
	parsed := parseRedisInfo(infoRaw)
	resp.Server = buildServerInfo(parsed)

	if resp.Usage != nil && resp.Server != nil {
		h := resp.Server.KeyspaceHits
		miss := resp.Server.KeyspaceMisses
		if h+miss > 0 {
			resp.Usage.HitRatioEstimate = float64(h) / float64(h+miss)
		}
	}

	n, err := client.DBSize(ctx).Result()
	if err != nil {
		if resp.Error == "" {
			resp.Error = "dbsize: " + err.Error()
		}
	} else {
		resp.KeysInDB = n
	}

	return resp
}
