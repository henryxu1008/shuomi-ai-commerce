package shein

import (
	"time"
)

type ResolutionCacheInfo struct {
	Status       string     `json:"status,omitempty"`
	Source       string     `json:"source,omitempty"`
	HitSource    string     `json:"hit_source,omitempty"`
	CacheKey     string     `json:"cache_key,omitempty"`
	ShortKey     string     `json:"short_key,omitempty"`
	HitCount     int        `json:"hit_count,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
	DisplayValue string     `json:"display_value,omitempty"`
	Manual       bool       `json:"manual,omitempty"`
	Clearable    bool       `json:"clearable,omitempty"`
}
