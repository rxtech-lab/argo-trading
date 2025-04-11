package cache

import (
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
	otherData        map[string]any
}

func NewCacheV1() Cache {
	return &CacheV1{
		RangeFilterState: optional.None[RangeFilterState](),
		otherData:        make(map[string]any),
	}
}

// Reset implements cache.Cache.
func (c *CacheV1) Reset() {
	c.RangeFilterState = optional.None[RangeFilterState]()
	c.otherData = make(map[string]any)
}

func (c *CacheV1) Set(key string, value any) {
	c.otherData[key] = value
}

func (c *CacheV1) Get(key string) (any, bool) {
	value, ok := c.otherData[key]
	return value, ok
}
