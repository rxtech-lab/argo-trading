package cache

import (
	"sync"

	"github.com/moznion/go-optional"
)

type Cache interface {
	Reset()
}

type RangeFilterState struct {
	Initialized bool    `json:"initialized"`
	PrevFilt    float64 `json:"prev_filt"`
	PrevSource  float64 `json:"prev_source"`
	Upward      float64 `json:"upward"`
	Downward    float64 `json:"downward"`
	Symbol      string  `json:"symbol"` // Symbol this state applies to
}

type CacheV1 struct {
	RangeFilterState optional.Option[RangeFilterState]
	mu               sync.RWMutex
}

func NewCacheV1() Cache {
	return &CacheV1{
		RangeFilterState: optional.None[RangeFilterState](),
	}
}

// Reset implements cache.Cache.
func (c *CacheV1) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.RangeFilterState = optional.None[RangeFilterState]()
}
