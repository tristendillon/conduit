package cache

import (
	"time"
)

type CacheConfig struct {
	MaxEntries      int           `json:"max_entries"`
	DefaultTTL      time.Duration `json:"default_ttl"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	EnableMetrics   bool          `json:"enable_metrics"`
}

func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		MaxEntries:      1000,
		DefaultTTL:      15 * time.Minute,
		CleanupInterval: time.Hour,
		EnableMetrics:   true,
	}
}

type CacheMetrics struct {
	Hits          int64   `json:"hits"`
	Misses        int64   `json:"misses"`
	Invalidations int64   `json:"invalidations"`
	TotalEntries  int     `json:"total_entries"`
	HitRate       float64 `json:"hit_rate"`
}

func (m *CacheMetrics) CalculateHitRate() {
	total := m.Hits + m.Misses
	if total > 0 {
		m.HitRate = float64(m.Hits) / float64(total) * 100
	} else {
		m.HitRate = 0
	}
}
