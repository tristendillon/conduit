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
	// Total metrics (including cold start)
	Hits          int64   `json:"hits"`
	Misses        int64   `json:"misses"`
	Invalidations int64   `json:"invalidations"`
	TotalEntries  int     `json:"total_entries"`
	HitRate       float64 `json:"hit_rate"`

	// Warm cache metrics (post-initialization)
	WarmHits    int64   `json:"warm_hits"`
	WarmMisses  int64   `json:"warm_misses"`
	WarmHitRate float64 `json:"warm_hit_rate"`

	// Cache state
	IsWarmed bool `json:"is_warmed"`
}

func (m *CacheMetrics) CalculateHitRate() {
	// Calculate total hit rate (including cold start)
	total := m.Hits + m.Misses
	if total > 0 {
		m.HitRate = float64(m.Hits) / float64(total) * 100
	} else {
		m.HitRate = 0
	}

	// Calculate warm hit rate (post-initialization only)
	warmTotal := m.WarmHits + m.WarmMisses
	if warmTotal > 0 {
		m.WarmHitRate = float64(m.WarmHits) / float64(warmTotal) * 100
	} else {
		m.WarmHitRate = 0
	}
}
