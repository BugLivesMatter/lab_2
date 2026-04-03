package cache

import (
	"sync"
	"time"
)

// accessMetrics хранит время последнего успешного обращения к Redis и счётчики операций.
type accessMetrics struct {
	mu             sync.RWMutex
	lastAccess     time.Time
	gets           uint64
	sets           uint64
	dels           uint64
	exists         uint64
	patternDeletes uint64
}

func newAccessMetrics() *accessMetrics {
	return &accessMetrics{}
}

func (m *accessMetrics) recordGet() {
	m.mu.Lock()
	m.lastAccess = time.Now().UTC()
	m.gets++
	m.mu.Unlock()
}

func (m *accessMetrics) recordSet() {
	m.mu.Lock()
	m.lastAccess = time.Now().UTC()
	m.sets++
	m.mu.Unlock()
}

func (m *accessMetrics) recordDel() {
	m.mu.Lock()
	m.lastAccess = time.Now().UTC()
	m.dels++
	m.mu.Unlock()
}

func (m *accessMetrics) recordExists() {
	m.mu.Lock()
	m.lastAccess = time.Now().UTC()
	m.exists++
	m.mu.Unlock()
}

func (m *accessMetrics) recordPatternDel() {
	m.mu.Lock()
	m.lastAccess = time.Now().UTC()
	m.patternDeletes++
	m.mu.Unlock()
}

func (m *accessMetrics) snapshot() (last time.Time, hasLast bool, gets, sets, dels, exists, patternDels uint64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.lastAccess.IsZero() {
		return time.Time{}, false, m.gets, m.sets, m.dels, m.exists, m.patternDeletes
	}
	return m.lastAccess, true, m.gets, m.sets, m.dels, m.exists, m.patternDeletes
}
